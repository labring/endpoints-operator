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
	"errors"
	"fmt"
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
	Probe(host string, port int, testData string, timeout time.Duration) (probe.Result, string, error)
}

type udpProber struct{}

// Probe returns a ProbeRunner capable of running a UDP check.
func (pr udpProber) Probe(host string, port int, testData string, timeout time.Duration) (probe.Result, string, error) {
	return DoUDPProbe(net.JoinHostPort(host, strconv.Itoa(port)), testData, timeout)
}

// DoUDPProbe checks that a UDP socket to the address can be opened.
// If the socket can be opened, it returns Success
// If the socket fails to open, it returns Failure.
// This is exported because some other packages may want to do direct UDP probes.
func DoUDPProbe(addr string, testData string, timeout time.Duration) (probe.Result, string, error) {

	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	klog.Infoln("Scan UDP Endpoint: ", addr)
	if err != nil {
		return probe.Failure, err.Error(), nil
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return probe.Failure, err.Error(), nil
	}
	defer conn.Close()
	deadline := time.Now().Add(timeout * time.Second)
	_ = conn.SetWriteDeadline(deadline)
	buf := []byte(testData)
	_, err1 := conn.Write(buf)
	if err1 != nil {
		return probe.Failure, err1.Error(), nil
	}
	_ = conn.SetReadDeadline(deadline)
	bufr := make([]byte, 1024)
	read, err2 := conn.Read(bufr)

	if err2 != nil {
		klog.Errorf("Unexpected error closing UDP probe socket: %v (%#v)", err2, err2)
		return probe.Failure, errors.New("io read timout").Error(), nil
	}
	if read > 0 {
		fmt.Println("recv:", string(bufr[:read]))
		return probe.Success, "", nil
	} else {
		return probe.Failure, err2.Error(), nil
	}
}
