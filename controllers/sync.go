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
	"github.com/sealyun/endpoints-operator/metrics"
	"strconv"
	"sync"

	"github.com/sealyun/endpoints-operator/api/network/v1beta1"
	libv1 "github.com/sealyun/endpoints-operator/library/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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
	var updateError error = nil
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ep := &corev1.Endpoints{}
		ep.SetName(cep.Name)
		ep.SetNamespace(cep.Namespace)
		_, err := controllerutil.CreateOrUpdate(ctx, c.Client, ep, func() error {
			ep.Labels = map[string]string{}
			if err := controllerutil.SetControllerReference(cep, ep, c.scheme); err != nil {
				return err
			}
			healthyHosts := make([]healthyHostAndPort, 0)
			e := make([]error, 0)
			for _, h := range cep.Spec.Hosts {
				healthyPorts, errors := healthyCheck(h, cep, c.RetryCount, c.MetricsInfo)
				if len(healthyPorts) > 0 {
					healthyHosts = append(healthyHosts, healthyHostAndPort{
						sps:  healthyPorts,
						host: h,
					})
				}
				subErr := ToAggregate(errors)
				if subErr != nil && len(subErr.Errors()) != 0 {
					e = append(e, fmt.Errorf(subErr.Error()))

				}

			}
			if len(healthyHosts) != 0 {
				subsets := make([]corev1.EndpointSubset, 0)
				for _, subset := range healthyHosts {
					subsets = append(subsets, subset.toEndpoint())
				}
				ep.Subsets = subsets
			} else {
				ep.Subsets = []corev1.EndpointSubset{}
			}
			if len(e) != 0 {
				updateError = ToAggregate(e)
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
	if updateError != nil {
		endpointCondition.LastHeartbeatTime = metav1.Now()
		endpointCondition.Status = corev1.ConditionFalse
		endpointCondition.Reason = "EndpointSyncPortError"
		endpointCondition.Message = updateError.Error()
		c.updateCondition(cep, endpointCondition)
		c.Logger.V(4).Info("error healthy endpoint", "name", cep.Name, "msg", updateError.Error())
		return
	}
	if !isConditionTrue(cep, v1beta1.SyncEndpointReady) {
		c.updateCondition(cep, endpointCondition)
	}
}

func healthyCheck(host string, cep *v1beta1.ClusterEndpoint, retry int, metricsinfo *metrics.MetricsInfo) ([]v1beta1.ServicePort, []error) {
	var wg sync.WaitGroup
	var mx sync.Mutex
	var data []v1beta1.ServicePort
	var errors []error
	checkListChan := make(chan struct {
		CepName           string
		NsName            string
		TargetHostAndPort string
		probe             string
	}, 200)

	for _, p := range cep.Spec.Ports {
		wg.Add(1)
		go func(port v1beta1.ServicePort) {
			defer wg.Done()
			defer mx.Unlock()
			if port.TimeoutSeconds == 0 {
				port.TimeoutSeconds = 1
			}
			if port.SuccessThreshold == 0 {
				port.SuccessThreshold = 1
			}
			if port.FailureThreshold == 0 {
				port.FailureThreshold = 3
			}
			pro := &libv1.Probe{
				TimeoutSeconds:   port.TimeoutSeconds,
				SuccessThreshold: port.SuccessThreshold,
				FailureThreshold: port.FailureThreshold,
			}
			if port.HTTPGet != nil {
				// add metrics point
				checkListChan <- struct {
					CepName           string
					NsName            string
					TargetHostAndPort string
					probe             string
				}{CepName: cep.Name, NsName: cep.Namespace, TargetHostAndPort: host + ":" + strconv.Itoa(int(port.TargetPort)), probe: "HTTP"}
				pro.HTTPGet = &libv1.HTTPGetAction{
					Path:        port.HTTPGet.Path,
					Port:        intstr.FromInt(int(port.TargetPort)),
					Host:        host,
					Scheme:      port.HTTPGet.Scheme,
					HTTPHeaders: port.HTTPGet.HTTPHeaders,
				}
			}
			if port.TCPSocket != nil && port.TCPSocket.Enable {
				// add metrics point
				checkListChan <- struct {
					CepName           string
					NsName            string
					TargetHostAndPort string
					probe             string
				}{CepName: cep.Name, NsName: cep.Namespace, TargetHostAndPort: host + ":" + strconv.Itoa(int(port.TargetPort)), probe: "TCP"}

				pro.TCPSocket = &libv1.TCPSocketAction{
					Port: intstr.FromInt(int(port.TargetPort)),
					Host: host,
				}
			}
			if port.UDPSocket != nil && port.UDPSocket.Enable {
				// add metrics point
				checkListChan <- struct {
					CepName           string
					NsName            string
					TargetHostAndPort string
					probe             string
				}{CepName: cep.Name, NsName: cep.Namespace, TargetHostAndPort: host + ":" + strconv.Itoa(int(port.TargetPort)), probe: "UDP"}

				pro.UDPSocket = &libv1.UDPSocketAction{
					Port: intstr.FromInt(int(port.TargetPort)),
					Host: host,
					Data: port.UDPSocket.Data,
				}
			}
			if port.GRPC != nil && port.GRPC.Enable {
				// add metrics point
				checkListChan <- struct {
					CepName           string
					NsName            string
					TargetHostAndPort string
					probe             string
				}{CepName: cep.Name, NsName: cep.Namespace, TargetHostAndPort: host + ":" + strconv.Itoa(int(port.TargetPort)), probe: "GRPC"}

				pro.GRPC = &libv1.GRPCAction{
					Port:    port.TargetPort,
					Host:    host,
					Service: port.GRPC.Service,
				}
			}
			w := &work{p: pro, retry: retry}
			for w.doProbe() {
			}
			mx.Lock()
			err := w.err
			if err != nil {
				// add metrics point
				metricsinfo.RecordFailedCheck(cep.Name, cep.Namespace, host+":"+strconv.Itoa(int(port.TargetPort)), w.checkProbe)
				errors = append(errors, err)
			} else {
				// add metrics point
				metricsinfo.RecordSuccessfulCheck(cep.Name, cep.Namespace, host+":"+strconv.Itoa(int(port.TargetPort)), w.checkProbe)
				data = append(data, port)
			}
		}(p)
	}
	wg.Wait()
	close(checkListChan)
	for checkdata := range checkListChan {
		metricsinfo.RecordCheck(checkdata.CepName, checkdata.NsName, checkdata.TargetHostAndPort, checkdata.probe)
		//metricsinfo.RecordCeps(checkdata.NsName)
	}
	return data, errors
}
