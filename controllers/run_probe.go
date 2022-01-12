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
	"fmt"
	"github.com/sealyun/endpoints-operator/library/probe"
	execprobe "github.com/sealyun/endpoints-operator/library/probe/exec"
	httpprobe "github.com/sealyun/endpoints-operator/library/probe/http"
	tcpprobe "github.com/sealyun/endpoints-operator/library/probe/tcp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Prober helps to check the liveness/readiness/startup of a container.
type prober struct {
	exec execprobe.Prober
	http httpprobe.Prober
	tcp  tcpprobe.Prober
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
	}
}

func (pb *prober) runProbe(p *v1.Probe) (probe.Result, string, error) {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if p.Exec != nil {
		klog.V(4).Infof("Exec-Probe Command: %v", p.Exec.Command)
		//command := ""
		return probe.Unknown, "", nil
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
	klog.Warning("Failed to find probe builder for annotation")
	return probe.Unknown, "", fmt.Errorf("missing probe handler")
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
