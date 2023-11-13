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
	"github.com/labring/endpoints-operator/utils/metrics"
	"reflect"
	"testing"

	"github.com/labring/endpoints-operator/apis/network/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_clusterEndpointConvertEndpointSubset(t *testing.T) {
	type args struct {
		cep         *v1beta1.ClusterEndpoint
		retry       int
		metricsinfo *metrics.MetricsInfo
	}
	tests := []struct {
		name  string
		args  args
		want  []corev1.EndpointSubset
		want1 []error
	}{
		{
			name: "default",
			args: args{
				cep: &v1beta1.ClusterEndpoint{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name:      "",
						Namespace: "",
					},
					Spec: v1beta1.ClusterEndpointSpec{
						Ports: []v1beta1.ServicePort{
							{
								Hosts: []string{"172.18.1.38", "172.18.1.69", "172.18.2.18"},
								Handler: v1beta1.Handler{
									TCPSocket: &v1beta1.TCPSocketAction{Enable: true},
								},
								TimeoutSeconds:   1,
								SuccessThreshold: 1,
								FailureThreshold: 1,
								Name:             "default",
								Protocol:         "TCP",
								Port:             8080,
								TargetPort:       31381,
							},
						},
						PeriodSeconds: 0,
					},
				},
				retry:       3,
				metricsinfo: nil,
			},
			want:  nil,
			want1: nil,
		},
		{
			name: "endpoint",
			args: args{
				cep: &v1beta1.ClusterEndpoint{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name:      "",
						Namespace: "",
					},
					Spec: v1beta1.ClusterEndpointSpec{
						Ports: []v1beta1.ServicePort{
							{
								Hosts: []string{"172.31.13.241", "172.31.3.240", "172.31.4.233"},
								Handler: v1beta1.Handler{
									HTTPGet: &v1beta1.HTTPGetAction{
										Path:   "/",
										Scheme: "HTTP",
									},
								},
								TimeoutSeconds:   1,
								SuccessThreshold: 1,
								FailureThreshold: 3,
								Name:             "default",
								Protocol:         "TCP",
								Port:             8848,
								TargetPort:       8848,
							},
						},
						PeriodSeconds: 0,
					},
				},
				retry:       3,
				metricsinfo: nil,
			},
			want:  nil,
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := clusterEndpointConvertEndpointSubset(tt.args.cep, tt.args.retry, tt.args.metricsinfo)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clusterEndpointConvertEndpointSubset() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("clusterEndpointConvertEndpointSubset() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
