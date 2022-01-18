/*
Copyright 2015 The Kubernetes Authors.

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

package udp

import (
	"github.com/sealyun/endpoints-operator/library/probe"
	"k8s.io/klog"
	"net"
	"strconv"
	"time"
)

// New creates Prober.
func New() Prober {
	return udpProber{}
}

// Prober is an interface that defines the Probe function for doing UDP readiness/liveness checks.
type Prober interface {
	Probe(host string, port int, timeout time.Duration) (probe.Result, string, error)
}

type udpProber struct{}

// Probe returns a ProbeRunner capable of running an UDP check.
func (pr udpProber) Probe(host string, port int, timeout time.Duration) (probe.Result, string, error) {
	return DoUDPProbe(net.JoinHostPort(host, strconv.Itoa(port)), timeout)
}

// DoUDPProbe checks that a UDP socket to the address can be opened.
// If the socket can be opened, it returns Success
// If the socket fails to open, it returns Failure.
// This is exported because some other packages may want to do direct TCP probes.
func DoUDPProbe(addr string, timeout time.Duration) (probe.Result, string, error) {
	conn, err := net.DialTimeout("udp", addr, timeout)
	if err != nil {
		// Convert errors to failures to handle timeouts.
		return probe.Failure, err.Error(), nil
	}
	err = conn.Close()
	if err != nil {
		klog.Errorf("Unexpected error closing UDP probe socket: %v (%#v)", err, err)
	}
	return probe.Success, "", nil
}
