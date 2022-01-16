/*
Copyright 2022 The sealyun Authors.

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
	"github.com/go-logr/logr"
	"github.com/sealyun/endpoints-operator/api/network/v1beta1"
	"github.com/sealyun/endpoints-operator/library/controller"
	"github.com/sealyun/endpoints-operator/library/convert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
	WorkNum   int
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
		Watches(&source.Kind{Type: &corev1.Service{}}, owner).WithOptions(runtimecontroller.Options{MaxConcurrentReconciles: c.WorkNum}).
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

func (c *Reconciler) Delete(ctx context.Context, req ctrl.Request, gvk schema.GroupVersionKind, obj runtime.Object) error {
	return nil
}
