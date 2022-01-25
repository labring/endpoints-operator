/*
Copyright 2022 The sealyun Authors.
Copyright 2014 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	libv1 "github.com/sealyun/endpoints-operator/library/api/core/v1"
	"github.com/sealyun/endpoints-operator/library/probe"
	execprobe "github.com/sealyun/endpoints-operator/library/probe/exec"
	grpcprobe "github.com/sealyun/endpoints-operator/library/probe/grpc"
	httpprobe "github.com/sealyun/endpoints-operator/library/probe/http"
	tcpprobe "github.com/sealyun/endpoints-operator/library/probe/tcp"
	udpprobe "github.com/sealyun/endpoints-operator/library/probe/udp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	urutime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

type work struct {
	p          *libv1.Probe
	resultRun  int
	lastResult probe.Result
	retry      int
	err        error
}

func (pb *prober) runProbeWithRetries(p *libv1.Probe, retries int) (probe.Result, string, error) {
	var err error
	var result probe.Result
	var output string
	for i := 0; i < retries; i++ {
		result, output, err = pb.runProbe(p)
		if err == nil {
			return result, output, nil
		}
	}
	return result, output, err
}

func (w *work) doProbe() (keepGoing bool) {
	defer func() { recover() }() // Actually eat panics (HandleCrash takes care of logging)
	defer urutime.HandleCrash(func(_ interface{}) { keepGoing = true })

	// the full container environment here, OR we must make a call to the CRI in order to get those environment
	// values from the running container.
	result, output, err := proberCheck.runProbeWithRetries(w.p, w.retry)
	if err != nil {
		w.err = err
		return false
	}

	if w.lastResult == result {
		w.resultRun++
	} else {
		w.lastResult = result
		w.resultRun = 1
	}

	if (result == probe.Failure && w.resultRun < int(w.p.FailureThreshold)) ||
		(result == probe.Success && w.resultRun < int(w.p.SuccessThreshold)) {
		// Success or failure is below threshold - leave the probe state unchanged.
		return true
	}

	if err != nil {
		w.err = err
	} else if result == probe.Failure && len(output) != 0 {
		w.err = errors.New(output)
	}
	return false
}

// Prober helps to check the liveness/readiness/startup of a container.
type prober struct {
	exec execprobe.Prober
	http httpprobe.Prober
	tcp  tcpprobe.Prober
	udp  udpprobe.Prober
	grpc grpcprobe.Prober
}

var proberCheck = newProber()

// NewProber creates a Prober, it takes a command runner and
// several container info managers.
func newProber() *prober {

	const followNonLocalRedirects = false
	return &prober{
		exec: execprobe.New(),
		http: httpprobe.New(followNonLocalRedirects),
		tcp:  tcpprobe.New(),
		udp:  udpprobe.New(),
		grpc: grpcprobe.New(),
	}
}

func (pb *prober) runProbe(p *libv1.Probe) (probe.Result, string, error) {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if p.Exec != nil {
		klog.V(4).Infof("Exec-Probe Command: %v", p.Exec.Command)
		//command := ""
		return probe.Success, "", nil
	}
	if p.HTTPGet != nil {
		scheme := strings.ToLower(string(p.HTTPGet.Scheme))
		host := p.HTTPGet.Host
		port, err := extractPort(p.HTTPGet.Port)
		if err != nil {
			return probe.Unknown, "", err
		}
		path := p.HTTPGet.Path
		klog.V(4).Infof("HTTP-Probe Host: %v://%v, Port: %v, Path: %v", scheme, host, port, path)
		url := formatURL(scheme, host, port, path)
		headers := buildHeader(p.HTTPGet.HTTPHeaders)
		klog.V(4).Infof("HTTP-Probe Headers: %v", headers)
		return pb.http.Probe(url, headers, timeout)
	}
	if p.TCPSocket != nil {
		port, err := extractPort(p.TCPSocket.Port)
		if err != nil {
			return probe.Unknown, "", err
		}
		host := p.TCPSocket.Host
		klog.V(4).Infof("TCP-Probe Host: %v, Port: %v, Timeout: %v", host, port, timeout)
		return pb.tcp.Probe(host, port, timeout)
	}
	if p.UDPSocket != nil {
		port, err := extractPort(p.UDPSocket.Port)
		if err != nil {
			return probe.Unknown, "", err
		}
		host := p.UDPSocket.Host
		klog.V(4).Infof("UDP-Probe Host: %v, Port: %v, Timeout: %v", host, port, timeout)
		return pb.udp.Probe(host, port, p.UDPSocket.Data, timeout)
	}
	if p.GRPC != nil {
		host := &(p.GRPC.Host)
		service := p.GRPC.Service
		klog.V(4).Info("GRPC-Probe Host: %v,Service: %v, Port: %v, Timeout: %v", host, service, p.GRPC.Port, timeout)
		return pb.grpc.Probe(*host, *service, int(p.GRPC.Port), timeout)
	}
	klog.Warning("failed to find probe builder")
	return probe.Warning, "", nil
}

func extractPort(param intstr.IntOrString) (int, error) {
	port := -1
	switch param.Type {
	case intstr.Int:
		port = param.IntValue()
	default:
		return port, fmt.Errorf("intOrString had no kind: %+v", param)
	}
	if port > 0 && port < 65536 {
		return port, nil
	}
	return port, fmt.Errorf("invalid port number: %v", port)
}

// formatURL formats a URL from args.  For testability.
func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	// Something is busted with the path, but it's too late to reject it. Pass it along as is.
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}

// buildHeaderMap takes a list of HTTPHeader <name, value> string
// pairs and returns a populated string->[]string http.Header map.
func buildHeader(headerList []v1.HTTPHeader) http.Header {
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}
