/*
Copyright 2022 cuisongliu@qq.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/sealyun/endpoints-operator/api/network/v1beta1"
	"github.com/sealyun/endpoints-operator/library/controller"
	"github.com/sealyun/endpoints-operator/library/convert"
	"github.com/sealyun/endpoints-operator/library/probe"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	urutime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

const (
	controllerName = "cluster_endpoints_controller"
)

// Reconciler reconciles a Service object
type Reconciler struct {
	client.Client
	Logger    logr.Logger
	Recorder  record.EventRecorder
	cache     cache.Cache
	scheme    *runtime.Scheme
	LoopCount int
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	rootCtx := context.Background()
	r.Logger.V(4).Info("start reconcile for ceps")
	ceps := &v1beta1.ClusterEndpoint{}
	ctr := controller.Controller{
		Client:   r.Client,
		Eventer:  r.Recorder,
		Operator: r,
		Gvk: schema.GroupVersionKind{
			Group:   v1beta1.GroupVersion.Group,
			Version: v1beta1.GroupVersion.Version,
			Kind:    "ClusterEndpoint",
		},
		FinalizerName: "sealyun.com/cluster-endpoints.finalizers",
	}
	ceps.APIVersion = ctr.Gvk.GroupVersion().String()
	ceps.Kind = ctr.Gvk.Kind
	return ctr.Run(rootCtx, req, ceps)
}

func (c *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if c.Client == nil {
		c.Client = mgr.GetClient()
	}
	if c.Logger == nil {
		c.Logger = log.Log.WithName(controllerName)
	}
	if c.Recorder == nil {
		c.Recorder = mgr.GetEventRecorderFor(controllerName)
	}
	c.scheme = mgr.GetScheme()
	c.cache = mgr.GetCache()
	c.Logger.V(4).Info("init reconcile controller service")
	owner := &handler.EnqueueRequestForOwner{OwnerType: &v1beta1.ClusterEndpoint{}, IsController: false}
	return ctrl.NewControllerManagedBy(mgr).WithEventFilter(&ResourceChangedPredicate{}).
		Watches(&source.Kind{Type: &corev1.Service{}}, owner).
		For(&v1beta1.ClusterEndpoint{}).Complete(c)
}

func (c *Reconciler) Update(ctx context.Context, req ctrl.Request, gvk schema.GroupVersionKind, obj runtime.Object) (ctrl.Result, error) {
	c.Logger.V(4).Info("update reconcile controller service", "request", req)
	cep := &v1beta1.ClusterEndpoint{}
	err := convert.JsonConvert(obj, cep)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	return c.UpdateStatus(ctx, req, cep)
}

func (c *Reconciler) UpdateStatus(ctx context.Context, req ctrl.Request, cep *v1beta1.ClusterEndpoint) (ctrl.Result, error) {
	initializedCondition := v1beta1.Condition{
		Type:               v1beta1.Initialized,
		Status:             corev1.ConditionTrue,
		Reason:             string(v1beta1.Initialized),
		Message:            "cluster endpoints has been initialized",
		LastHeartbeatTime:  metav1.Now(),
		LastTransitionTime: metav1.Now(),
	}
	cep.Status.Phase = v1beta1.Pending
	if !isConditionTrue(cep, v1beta1.Initialized) {
		c.updateCondition(cep, initializedCondition)
	}

	c.syncService(ctx, cep)
	c.syncEndpoint(ctx, cep)

	c.Logger.V(4).Info("update finished reconcile controller service", "request", req)
	c.syncFinalStatus(cep)
	err := c.updateStatus(ctx, req.NamespacedName, &cep.Status)
	if err != nil {
		c.Recorder.Eventf(cep, corev1.EventTypeWarning, "SyncStatus", "Sync status %s is error: %v", cep.Name, err)
	}
	sec := time.Duration(cep.Spec.PeriodSeconds) * time.Second
	if cep.Spec.PeriodSeconds == 0 {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: sec}, nil
}

func (c *Reconciler) syncService(ctx context.Context, cep *v1beta1.ClusterEndpoint) {
	serviceCondition := v1beta1.Condition{
		Type:               v1beta1.SyncServiceReady,
		Status:             corev1.ConditionTrue,
		LastHeartbeatTime:  metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             string(v1beta1.SyncServiceReady),
		Message:            "sync service successfully",
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		svc := &corev1.Service{}
		svc.SetName(cep.Name)
		svc.SetNamespace(cep.Namespace)
		_, err := controllerutil.CreateOrUpdate(ctx, c.Client, svc, func() error {
			svc.Labels = map[string]string{}
			if err := controllerutil.SetControllerReference(cep, svc, c.scheme); err != nil {
				return err
			}
			svc.Spec.ClusterIP = cep.Spec.ClusterIP
			svc.Spec.Type = corev1.ServiceTypeClusterIP
			svc.Spec.SessionAffinity = corev1.ServiceAffinityNone
			svc.Spec.Ports = convertServicePorts(cep.Spec.Ports)
			return nil
		})
		return err
	}); err != nil {
		serviceCondition.LastHeartbeatTime = metav1.Now()
		serviceCondition.Status = corev1.ConditionFalse
		serviceCondition.Reason = "ServiceSyncError"
		serviceCondition.Message = err.Error()
		c.updateCondition(cep, serviceCondition)
		c.Logger.V(4).Info("error updating service", "name", cep.Name, "msg", err.Error())
		return
	}
	if !isConditionTrue(cep, v1beta1.SyncServiceReady) {
		c.updateCondition(cep, serviceCondition)
	}
}
func (c *Reconciler) syncEndpoint(ctx context.Context, cep *v1beta1.ClusterEndpoint) {
	endpointCondition := v1beta1.Condition{
		Type:               v1beta1.SyncEndpointReady,
		Status:             corev1.ConditionTrue,
		LastHeartbeatTime:  metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             string(v1beta1.SyncEndpointReady),
		Message:            "sync endpoint successfully",
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ep := &corev1.Endpoints{}
		ep.SetName(cep.Name)
		ep.SetNamespace(cep.Namespace)
		_, err := controllerutil.CreateOrUpdate(ctx, c.Client, ep, func() error {
			ep.Labels = map[string]string{}
			healthyHosts := make([]string, 0)
			e := make([]string, 0)
			for _, h := range cep.Spec.Hosts {
				if err := healthyCheck(ctx, h, cep); err == nil {
					healthyHosts = append(healthyHosts, h)
				} else {
					e = append(e, fmt.Sprintf("host: %s heathy is unhealthy: %v ", h, err.Error()))
				}
			}
			hosts := convertAddress(healthyHosts)
			if len(hosts) != 0 {
				es := corev1.EndpointSubset{
					Addresses: hosts,
					Ports:     convertPorts(cep.Spec.Ports),
				}
				ep.Subsets = []corev1.EndpointSubset{es}
			}
			if len(e) != 0 {
				return fmt.Errorf(strings.Join(e, ";;"))
			}
			return nil
		})
		return err
	}); err != nil {
		endpointCondition.LastHeartbeatTime = metav1.Now()
		endpointCondition.Status = corev1.ConditionFalse
		endpointCondition.Reason = "EndpointSyncError"
		endpointCondition.Message = err.Error()
		c.updateCondition(cep, endpointCondition)
		c.Logger.V(4).Info("error updating endpoint", "name", cep.Name, "msg", err.Error())
		return
	}
	if !isConditionTrue(cep, v1beta1.SyncEndpointReady) {
		c.updateCondition(cep, endpointCondition)
	}
}

func healthyCheck(ctx context.Context, host string, cep *v1beta1.ClusterEndpoint) error {
	pg, _ := errgroup.WithContext(ctx)
	for _, p := range cep.Spec.Ports {
		p := p
		pg.Go(func() error {
			if p.TimeoutSeconds == 0 {
				p.TimeoutSeconds = 1
			}
			if p.SuccessThreshold == 0 {
				p.SuccessThreshold = 1
			}
			if p.FailureThreshold == 0 {
				p.FailureThreshold = 3
			}
			pro := &corev1.Probe{
				TimeoutSeconds:   p.TimeoutSeconds,
				SuccessThreshold: p.SuccessThreshold,
				FailureThreshold: p.FailureThreshold,
			}
			if p.HTTPGet != nil {
				pro.HTTPGet = &corev1.HTTPGetAction{
					Path:        p.HTTPGet.Path,
					Port:        intstr.FromInt(int(p.TargetPort)),
					Host:        host,
					Scheme:      p.HTTPGet.Scheme,
					HTTPHeaders: p.HTTPGet.HTTPHeaders,
				}
			}
			if p.TCPSocket != nil && p.TCPSocket.Enable {
				pro.TCPSocket = &corev1.TCPSocketAction{
					Port: intstr.FromInt(int(p.TargetPort)),
					Host: host,
				}
			}
			w := &work{p: pro}
			for w.doProbe() {
			}
			return w.err
		})
	}
	return pg.Wait()
}

type work struct {
	p          *corev1.Probe
	resultRun  int
	lastResult probe.Result
	err        error
}

func (pb *prober) runProbeWithRetries(p *corev1.Probe, retries int) (probe.Result, string, error) {
	var err error
	var result probe.Result
	var output string
	for i := 0; i < retries; i++ {
		result, output, err = pb.runProbe(p)
		if err == nil {
			return result, output, nil
		}
	}
	return result, output, err
}

func (w *work) doProbe() (keepGoing bool) {
	defer func() { recover() }() // Actually eat panics (HandleCrash takes care of logging)
	defer urutime.HandleCrash(func(_ interface{}) { keepGoing = true })

	// the full container environment here, OR we must make a call to the CRI in order to get those environment
	// values from the running container.
	result, output, err := proberCheck.runProbeWithRetries(w.p, 3)
	if err != nil {
		w.err = err
		return false
	}

	if w.lastResult == result {
		w.resultRun++
	} else {
		w.lastResult = result
		w.resultRun = 1
	}

	if (result == probe.Failure && w.resultRun < int(w.p.FailureThreshold)) ||
		(result == probe.Success && w.resultRun < int(w.p.SuccessThreshold)) {
		// Success or failure is below threshold - leave the probe state unchanged.
		return true
	}
	if err != nil {
		w.err = err
	} else if len(output) != 0 {
		w.err = errors.New(output)
	}
	return false
}

func (c *Reconciler) updateStatus(ctx context.Context, nn types.NamespacedName, status *v1beta1.ClusterEndpointStatus) error {
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		original := &v1beta1.ClusterEndpoint{}
		if err := c.Get(ctx, nn, original); err != nil {
			return err
		}
		original.Status = *status
		if err := c.Client.Status().Update(ctx, original); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
func (c *Reconciler) syncFinalStatus(cep *v1beta1.ClusterEndpoint) {
	clusterReadyCondition := v1beta1.Condition{
		Type:               v1beta1.Ready,
		Status:             corev1.ConditionTrue,
		LastHeartbeatTime:  metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             string(v1beta1.Ready),
		Message:            "ClusterEndpoint is available now",
	}
	if isConditionsTrue(cep) {
		cep.Status.Phase = v1beta1.Healthy
	} else {
		clusterReadyCondition.LastHeartbeatTime = metav1.Now()
		clusterReadyCondition.Status = corev1.ConditionFalse
		clusterReadyCondition.Reason = "Not" + string(v1beta1.Ready)
		clusterReadyCondition.Message = "ClusterEndpoint is not available now"
		cep.Status.Phase = v1beta1.UnHealthy
	}
	c.updateCondition(cep, clusterReadyCondition)
}

// updateCondition updates condition in cluster conditions using giving condition
// adds condition if not existed
func (c *Reconciler) updateCondition(cep *v1beta1.ClusterEndpoint, condition v1beta1.Condition) {
	if cep.Status.Conditions == nil {
		cep.Status.Conditions = make([]v1beta1.Condition, 0)
	}
	hasCondition := false
	for i, cond := range cep.Status.Conditions {
		if cond.Type == condition.Type {
			hasCondition = true
			if cond.Reason != condition.Reason || cond.Status != condition.Status {
				cep.Status.Conditions[i] = condition
			}
		}
	}
	if !hasCondition {
		cep.Status.Conditions = append(cep.Status.Conditions, condition)
	}
}
func (c *Reconciler) deleteCondition(cep *v1beta1.ClusterEndpoint, conditionType v1beta1.ConditionType) {
	if cep.Status.Conditions == nil {
		cep.Status.Conditions = make([]v1beta1.Condition, 0)
	}
	newConditions := make([]v1beta1.Condition, 0)
	for _, cond := range cep.Status.Conditions {
		if cond.Type == conditionType {
			continue
		}
		newConditions = append(newConditions, cond)
	}
	cep.Status.Conditions = newConditions
}

func convertAddress(addresses []string) []corev1.EndpointAddress {
	eas := make([]corev1.EndpointAddress, 0)
	for _, s := range addresses {
		eas = append(eas, corev1.EndpointAddress{
			IP: s,
		})
	}
	return eas
}

func convertPorts(sps []v1beta1.ServicePort) []corev1.EndpointPort {
	s := make([]corev1.EndpointPort, 0)
	for _, sp := range sps {
		endPoint := corev1.EndpointPort{
			Name:     sp.Name,
			Port:     sp.TargetPort,
			Protocol: sp.Protocol,
		}
		s = append(s, endPoint)
	}
	return s
}

func convertServicePorts(sps []v1beta1.ServicePort) []corev1.ServicePort {
	s := make([]corev1.ServicePort, 0)
	for _, sp := range sps {
		endPoint := corev1.ServicePort{
			Name:       sp.Name,
			Port:       sp.Port,
			Protocol:   sp.Protocol,
			TargetPort: intstr.FromInt(int(sp.TargetPort)),
		}
		s = append(s, endPoint)
	}
	return s
}

func (c *Reconciler) Delete(ctx context.Context, req ctrl.Request, gvk schema.GroupVersionKind, obj runtime.Object) error {
	return nil
}
