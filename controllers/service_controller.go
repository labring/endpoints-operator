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
	"github.com/cuisongliu/endpoints-balance/library/controller"
	"github.com/cuisongliu/endpoints-balance/library/convert"
	"github.com/cuisongliu/endpoints-balance/library/hash"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	cruntimecontrl "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controllerName = "service_controller"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Logger    logr.Logger
	Recorder  record.EventRecorder
	cache     cache.Cache
	LoopCount int
}

func (r *ServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	rootCtx := context.Background()
	r.Logger.V(4).Info("start reconcile for service")
	service := &corev1.Service{}
	ctr := controller.Controller{
		Client:   r.Client,
		Eventer:  r.Recorder,
		Operator: r,
		Gvk: schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Service",
		},
		FinalizerName: "sealyun.com/endpoint-balance.finalizers",
	}
	service.APIVersion = ctr.Gvk.GroupVersion().String()
	service.Kind = ctr.Gvk.Kind
	return ctr.Run(rootCtx, req, service)
}

func (c *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if c.Client == nil {
		c.Client = mgr.GetClient()
	}
	if c.Logger == nil {
		c.Logger = log.Log.WithName(controllerName)
	}
	if c.Recorder == nil {
		c.Recorder = mgr.GetEventRecorderFor(controllerName)
	}
	c.cache = mgr.GetCache()
	c.Logger.V(4).Info("init reconcile controller service")
	return ctrl.NewControllerManagedBy(mgr).WithOptions(cruntimecontrl.Options{MaxConcurrentReconciles: 1}).
		For(&corev1.Service{}).WithEventFilter(predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return c.processService(event.Meta)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return c.processService(deleteEvent.Meta)
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			if c.processService(updateEvent.MetaOld) {
				if newService, ok := updateEvent.ObjectNew.(*corev1.Service); ok {
					if oldService, ok := updateEvent.ObjectOld.(*corev1.Service); ok {
						if newService.DeletionTimestamp != nil {
							return true
						}
						newhash := hash.Hash(newService.Annotations)
						oldhash := hash.Hash(oldService.Annotations)
						if newhash != oldhash {
							return true
						}
					}
				}
			}
			return false
		},
	}).Complete(c)
}

func (r *ServiceReconciler) processService(meta metav1.Object) bool {
	if meta.GetAnnotations() != nil && meta.GetAnnotations()[annotationServer] != "" {
		return true
	}
	return false
}

func (c *ServiceReconciler) Update(ctx context.Context, req ctrl.Request, gvk schema.GroupVersionKind, obj runtime.Object) (ctrl.Result, error) {
	c.Logger.V(4).Info("update reconcile controller service", "request", req)
	svc := &corev1.Service{}
	err := convert.JsonConvert(obj, svc)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	ep := &corev1.Endpoints{}
	ep.SetName(svc.Name)
	ep.SetNamespace(svc.Namespace)
	annotationServers := svc.Annotations[annotationServer]
	ports := svc.Spec.Ports
	_, err = controllerutil.CreateOrUpdate(ctx, c.Client, ep, func() error {
		ep.Labels = map[string]string{}

		ep.Subsets = make([]corev1.EndpointSubset, 0)
		es := corev1.EndpointSubset{
			Addresses: convertAddress(annotationServers),
			Ports:     convertPorts(ports),
		}
		ep.Subsets = append(ep.Subsets, es)
		return nil
	})
	if err != nil {
		c.Logger.Error(err, "endpoint create or update failed")
		return ctrl.Result{Requeue: true}, err
	}
	c.Logger.V(4).Info("update finished reconcile controller service", "request", req)
	return ctrl.Result{}, nil
}

func convertAddress(address string) []corev1.EndpointAddress {
	servers := strings.Split(address, ",")
	eas := make([]corev1.EndpointAddress, 0)
	for _, s := range servers {
		eas = append(eas, corev1.EndpointAddress{
			IP: s,
		})
	}
	return eas
}

func convertPorts(sps []corev1.ServicePort) []corev1.EndpointPort {
	s := make([]corev1.EndpointPort, 0)
	for _, sp := range sps {
		endPoint := corev1.EndpointPort{
			Name:     sp.Name,
			Port:     sp.TargetPort.IntVal,
			Protocol: sp.Protocol,
		}
		s = append(s, endPoint)
	}
	return s
}

func (c *ServiceReconciler) Delete(ctx context.Context, req ctrl.Request, gvk schema.GroupVersionKind, obj runtime.Object) error {
	return nil
}
