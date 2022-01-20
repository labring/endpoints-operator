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
	"strings"

	"github.com/go-logr/logr"
	"github.com/sealyun/endpoints-operator/library/convert"
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
		return r.Operator.Update(ctx, req, r.Gvk, ustructObj)
	} else {
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
			controllerutil.RemoveFinalizer(ustructObj, r.FinalizerName)
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
}
