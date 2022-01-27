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

package v1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// HTTPHeader describes a custom header to be used in HTTP probes
type HTTPHeader struct {
	// The header field name
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// The header field value
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

// HTTPGetAction describes an action based on HTTP Get requests.
type HTTPGetAction struct {
	// Path to access on the HTTP server.
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	// Name or number of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port intstr.IntOrString `json:"port" protobuf:"bytes,2,opt,name=port"`
	// Host name to connect to, defaults to the pod IP. You probably want to set
	// "Host" in httpHeaders instead.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,3,opt,name=host"`
	// Scheme to use for connecting to the host.
	// Defaults to HTTP.
	// +optional
	Scheme v1.URIScheme `json:"scheme,omitempty" protobuf:"bytes,4,opt,name=scheme,casttype=URIScheme"`
	// Custom headers to set in the request. HTTP allows repeated headers.
	// +optional
	HTTPHeaders []v1.HTTPHeader `json:"httpHeaders,omitempty" protobuf:"bytes,5,rep,name=httpHeaders"`
}

// TCPSocketAction describes an action based on opening a socket
type TCPSocketAction struct {
	// Number or name of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port intstr.IntOrString `json:"port" protobuf:"bytes,1,opt,name=port"`
	// Optional: Host name to connect to, defaults to the pod IP.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,2,opt,name=host"`
}

// UDPSocketAction describes an action based on opening a socket
type UDPSocketAction struct {
	// Number or name of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port intstr.IntOrString `json:"port" protobuf:"bytes,1,opt,name=port"`
	// Optional: Host name to connect to, defaults to the pod IP.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,2,opt,name=host"`
	// +optional
	Data []byte `json:"data,omitempty" protobuf:"bytes,3,rep,name=data"`
}

type GRPCAction struct {
	// Port number of the gRPC service. Number must be in the range 1 to 65535.
	Port int32 `json:"port" protobuf:"bytes,1,opt,name=port"`

	// Service is the name of the service to place in the gRPC HealthCheckRequest
	// (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).
	//
	// If this is not specified, the default behavior is defined by gRPC.
	// +optional
	// +default=""
	Service *string `json:"service" protobuf:"bytes,2,opt,name=service"`
	// Optional: Host name to connect to, defaults to the pod IP.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,3,opt,name=host"`
}

// ExecAction describes a "run in container" action.
type ExecAction struct {
	// Command is the command line to execute inside the container, the working directory for the
	// command  is root ('/') in the container's filesystem. The command is simply exec'd, it is
	// not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use
	// a shell, you need to explicitly call out to that shell.
	// Exit status of 0 is treated as live/healthy and non-zero is unhealthy.
	// +optional
	Command []string `json:"command,omitempty" protobuf:"bytes,1,rep,name=command"`
}

// ProbeHandler defines a specific action that should be taken in a probe.
// One and only one of the fields must be specified.
type ProbeHandler struct {
	// Exec specifies the action to take.
	// +optional
	Exec *ExecAction `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`
	// HTTPGet specifies the http request to perform.
	// +optional
	HTTPGet *HTTPGetAction `json:"httpGet,omitempty" protobuf:"bytes,2,opt,name=httpGet"`
	// TCPSocket specifies an action involving a TCP port.
	// +optional
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty" protobuf:"bytes,3,opt,name=tcpSocket"`
	// UDPSocket specifies an action involving a UDP port.
	// +optional
	UDPSocket *UDPSocketAction `json:"udpSocket,omitempty" protobuf:"bytes,4,opt,name=udpSocket"`

	// GRPC specifies an action involving a GRPC port.
	// This is an alpha field and requires enabling GRPCContainerProbe feature gate.
	// +featureGate=GRPCContainerProbe
	// +optional
	GRPC *GRPCAction `json:"grpc,omitempty" protobuf:"bytes,5,opt,name=grpc"`
}

// Probe describes a health check to be performed against a container to determine whether it is
// alive or ready to receive traffic.
type Probe struct {
	// The action taken to determine the health of a container
	ProbeHandler `json:",inline" protobuf:"bytes,1,opt,name=handler"`
	// Number of seconds after the container has started before liveness probes are initiated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" protobuf:"varint,2,opt,name=initialDelaySeconds"`
	// Number of seconds after which the probe times out.
	// Defaults to 1 second. Minimum value is 1.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,3,opt,name=timeoutSeconds"`
	// How often (in seconds) to perform the probe.
	// Default to 10 seconds. Minimum value is 1.
	// +optional
	PeriodSeconds int32 `json:"periodSeconds,omitempty" protobuf:"varint,4,opt,name=periodSeconds"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.
	// +optional
	SuccessThreshold int32 `json:"successThreshold,omitempty" protobuf:"varint,5,opt,name=successThreshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Defaults to 3. Minimum value is 1.
	// +optional
	FailureThreshold int32 `json:"failureThreshold,omitempty" protobuf:"varint,6,opt,name=failureThreshold"`
	// Optional duration in seconds the pod needs to terminate gracefully upon probe failure.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this
	// value overrides the value provided by the pod spec.
	// Value must be non-negative integer. The value zero indicates stop immediately via
	// the kill signal (no opportunity to shut down).
	// This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate.
	// Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty" protobuf:"varint,7,opt,name=terminationGracePeriodSeconds"`
}
