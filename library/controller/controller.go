// Copyright © 2022 The sealyun Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"strings"

	"github.com/go-logr/logr"
	"github.com/labring/endpoints-operator/library/convert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Controller struct {
	client.Client
	Eventer       record.EventRecorder
	Operator      Operator
	Gvk           schema.GroupVersionKind
	Logger        logr.Logger
	FinalizerName string
}

var WaitDelete = fmt.Errorf("wait delete resource")

type Operator interface {
	Update(ctx context.Context, req ctrl.Request, gvk schema.GroupVersionKind, obj client.Object) (ctrl.Result, error)
	Delete(ctx context.Context, req ctrl.Request, gvk schema.GroupVersionKind, obj client.Object) error
}

func (r *Controller) GroupVersionKind() schema.GroupVersionKind {
	return r.Gvk
}
func (r *Controller) Run(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, error) {
	lowerKind := strings.ToLower(r.Gvk.Kind)
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		//
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ustructObj, err := convert.ResourceToUnstructured(obj)
	if err != nil {
		r.Logger.Error(err, "unable to convert", "kind", lowerKind)
		return ctrl.Result{Requeue: true}, err
	}
	if ustructObj.GetDeletionTimestamp() == nil || ustructObj.GetDeletionTimestamp().IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		controllerutil.AddFinalizer(ustructObj, r.FinalizerName)
		if err = r.setFinalizers(ctx, req, obj, ustructObj.GetFinalizers()); err != nil {
			r.Eventer.Eventf(obj, corev1.EventTypeWarning, "FailedUpdate", "Update %s: %v", lowerKind, err)
			//如果修改失败重新放入队列
			r.Logger.Error(err, "unable to set finalizer", "finalizer", r.FinalizerName)
			return ctrl.Result{Requeue: true}, err
		}
		return r.Operator.Update(ctx, req, r.Gvk, ustructObj)
	} else {
		r.Logger.V(4).Info("delete reconcile controller service", "request", req)
		// The object is being deleted
		if controllerutil.ContainsFinalizer(ustructObj, r.FinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err = r.Operator.Delete(ctx, req, r.Gvk, ustructObj); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				if err == WaitDelete {
					r.Logger.V(5).Info("wait gc the resource", "err", err.Error())
					return ctrl.Result{Requeue: true}, nil
				}
				r.Logger.Error(err, "failed delete the resource", "err", err.Error())
				r.Eventer.Eventf(obj, corev1.EventTypeWarning, "FailedDelete", "Deleted %s: %v", lowerKind, err)
				//如果修改失败重新放入队列
				return ctrl.Result{Requeue: true}, err
			}
			r.Logger.V(4).Info("remove finalizer  to delete obj", "finalizers", ustructObj.GetFinalizers())
			controllerutil.RemoveFinalizer(ustructObj, r.FinalizerName)
			if err = r.setFinalizers(ctx, req, obj, ustructObj.GetFinalizers()); err != nil {
				r.Eventer.Eventf(obj, corev1.EventTypeWarning, "FailedDelete", "Deleted %s: %v", lowerKind, err)
				r.Logger.Error(err, "failed set finalizer the resource", "err", err.Error())
				return ctrl.Result{Requeue: true}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
}

func (r *Controller) setFinalizers(ctx context.Context, req ctrl.Request, obj runtime.Object, finalizers []string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		gvk := obj.GetObjectKind().GroupVersionKind()
		fetchObject := &unstructured.Unstructured{}
		fetchObject.SetAPIVersion(gvk.GroupVersion().String())
		fetchObject.SetKind(gvk.Kind)
		err := r.Client.Get(ctx, req.NamespacedName, fetchObject)
		if err != nil {
			// We log this error, but we continue and try to set the ownerRefs on the other resources.
			return err
		}
		fetchObject.SetFinalizers(finalizers)
		err = r.Client.Update(ctx, fetchObject)
		if err != nil {
			// We log this error, but we continue and try to set the ownerRefs on the other resources.
			return err
		}
		return nil
	})
}
