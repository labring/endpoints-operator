/*
Copyright 2022 The sealos Authors.

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
	"time"

	"github.com/labring/endpoints-operator/metrics"

	"github.com/go-logr/logr"
	"github.com/labring/endpoints-operator/apis/network/v1beta1"
	"github.com/labring/endpoints-operator/library/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "cluster_endpoints_controller"
)

// Reconciler reconciles a Service object
type Reconciler struct {
	client.Client
	Logger      logr.Logger
	Recorder    record.EventRecorder
	cache       cache.Cache
	scheme      *runtime.Scheme
	RetryCount  int
	WorkNum     int
	MetricsInfo *metrics.MetricsInfo
	finalizer   *controller.Finalizer
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger.V(4).Info("start reconcile for ceps")
	cep := &v1beta1.ClusterEndpoint{}
	if err := r.Get(ctx, req.NamespacedName, cep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if ok, err := r.finalizer.RemoveFinalizer(ctx, cep, controller.DefaultFunc); ok {
		return ctrl.Result{}, err
	}

	if ok, err := r.finalizer.AddFinalizer(ctx, cep); ok {
		if err != nil {
			return ctrl.Result{}, err
		} else {
			return r.Update(ctx, cep)
		}
	}
	return ctrl.Result{}, errors.New("reconcile error from Finalizer")
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
	if c.finalizer == nil {
		c.finalizer = controller.NewFinalizer(c.Client, "sealos.io/cluster-endpoints.finalizers")
	}
	c.scheme = mgr.GetScheme()
	c.cache = mgr.GetCache()
	c.Logger.V(4).Info("init reconcile controller service")
	owner := &handler.EnqueueRequestForOwner{OwnerType: &v1beta1.ClusterEndpoint{}, IsController: false}
	return ctrl.NewControllerManagedBy(mgr).WithEventFilter(&ResourceChangedPredicate{}).
		Watches(&source.Kind{Type: &corev1.Service{}}, owner).WithOptions(runtimecontroller.Options{MaxConcurrentReconciles: c.WorkNum}).
		For(&v1beta1.ClusterEndpoint{}).Complete(c)
}

func (c *Reconciler) Update(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	c.Logger.V(4).Info("update reconcile controller service", "request", client.ObjectKeyFromObject(obj))
	cep, ok := obj.(*v1beta1.ClusterEndpoint)
	if !ok {
		return ctrl.Result{}, errors.New("obj convert cep is error")
	}

	return c.UpdateStatus(ctx, cep)
}

func (c *Reconciler) UpdateStatus(ctx context.Context, cep *v1beta1.ClusterEndpoint) (ctrl.Result, error) {
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

	c.Logger.V(4).Info("update finished reconcile controller service", "request", client.ObjectKeyFromObject(cep))
	c.syncFinalStatus(cep)
	err := c.updateStatus(ctx, client.ObjectKeyFromObject(cep), &cep.Status)
	if err != nil {
		c.Recorder.Eventf(cep, corev1.EventTypeWarning, "SyncStatus", "Sync status %s is error: %v", cep.Name, err)
		return ctrl.Result{}, err
	}
	sec := time.Duration(cep.Spec.PeriodSeconds) * time.Second
	if cep.Spec.PeriodSeconds == 0 {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: sec}, nil
}
