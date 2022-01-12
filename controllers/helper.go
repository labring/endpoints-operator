/*
Copyright 2020 KubeSphere Authors

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
	"github.com/sealyun/endpoints-operator/api/network/v1beta1"
	v1 "k8s.io/api/core/v1"
)

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
		if condition.Status != v1.ConditionTrue {
			return false
		}
	}
	return true
}
