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
	"fmt"

	"github.com/sealyun/endpoints-operator/api/network/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
)

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
		Status:             v1.ConditionTrue,
		LastHeartbeatTime:  metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             string(v1beta1.Ready),
		Message:            "ClusterEndpoint is available now",
	}
	if isConditionsTrue(cep) {
		cep.Status.Phase = v1beta1.Healthy
	} else {
		clusterReadyCondition.LastHeartbeatTime = metav1.Now()
		clusterReadyCondition.Status = v1.ConditionFalse
		clusterReadyCondition.Reason = "Not" + string(v1beta1.Ready)
		clusterReadyCondition.Message = "ClusterEndpoint is not available now"
		cep.Status.Phase = v1beta1.UnHealthy
	}
	c.updateCondition(cep, clusterReadyCondition)
}

func convertAddress(addresses []string) []v1.EndpointAddress {
	eas := make([]v1.EndpointAddress, 0)
	for _, s := range addresses {
		eas = append(eas, v1.EndpointAddress{
			IP: s,
		})
	}
	return eas
}

type healthyHostAndPort struct {
	sps  []v1beta1.ServicePort
	host string
}

func (hap *healthyHostAndPort) toEndpoint() v1.EndpointSubset {
	s := make([]v1.EndpointPort, 0)
	for _, sp := range hap.sps {
		endPoint := v1.EndpointPort{
			Name:     sp.Name,
			Port:     sp.TargetPort,
			Protocol: sp.Protocol,
		}
		s = append(s, endPoint)
	}
	return v1.EndpointSubset{
		Addresses: []v1.EndpointAddress{
			{
				IP: hap.host,
			},
		},
		Ports: s,
	}
}

// ToAggregate converts the ErrorList into an errors.Aggregate.
func ToAggregate(list []error) utilerrors.Aggregate {
	errs := make([]error, 0, len(list))
	errorMsgs := sets.NewString()
	for _, err := range list {
		msg := fmt.Sprintf("%v", err)
		if errorMsgs.Has(msg) {
			continue
		}
		errorMsgs.Insert(msg)
		errs = append(errs, err)
	}
	return utilerrors.NewAggregate(errs)
}

func convertPorts(sps []v1beta1.ServicePort) []v1.EndpointPort {
	s := make([]v1.EndpointPort, 0)
	for _, sp := range sps {
		endPoint := v1.EndpointPort{
			Name:     sp.Name,
			Port:     sp.TargetPort,
			Protocol: sp.Protocol,
		}
		s = append(s, endPoint)
	}
	return s
}

func convertServicePorts(sps []v1beta1.ServicePort) []v1.ServicePort {
	s := make([]v1.ServicePort, 0)
	for _, sp := range sps {
		endPoint := v1.ServicePort{
			Name:       sp.Name,
			Port:       sp.Port,
			Protocol:   sp.Protocol,
			TargetPort: intstr.FromInt(int(sp.TargetPort)),
		}
		s = append(s, endPoint)
	}
	return s
}

func isConditionTrue(ce *v1beta1.ClusterEndpoint, conditionType v1beta1.ConditionType) bool {
	for _, condition := range ce.Status.Conditions {
		if condition.Type == conditionType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}
func isConditionsTrue(ce *v1beta1.ClusterEndpoint) bool {
	if len(ce.Status.Conditions) == 0 {
		return false
	}
	for _, condition := range ce.Status.Conditions {
		if condition.Type == v1beta1.Ready {
			continue
		}
		if condition.Status != v1.ConditionTrue {
			return false
		}
	}
	return true
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
			if cond.Reason != condition.Reason || cond.Status != condition.Status || cond.Message != condition.Message {
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
