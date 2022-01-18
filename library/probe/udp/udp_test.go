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
	"fmt"
	"github.com/sealyun/endpoints-operator/library/probe"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
)

func udp() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("Can't resolve address: ", err)
		os.Exit(1)
	}
	conn, err := net.ListenUDP("udp", addr)
	//for {
	//	handleClient(conn)
	//}
	return conn, err
}

//func handleClient(conn *net.UDPConn) {
//	data := make([]byte, 1024)
//	n, remoteAddr, err := conn.ReadFromUDP(data)
//	if err != nil {
//		fmt.Println("failed to read UDP msg because of ", err.Error())
//		return
//	}
//	daytime := time.Now().Unix()
//	fmt.Println(n, remoteAddr)
//	b := make([]byte, 4)
//	binary.BigEndian.PutUint32(b, uint32(daytime))
//	conn.WriteToUDP(b, remoteAddr)
//}

func TestUDPHealthChecker(t *testing.T) {
	// Setup a test server that responds to probing correctly
	server, err := udp()
	defer server.Close()
	tHost, tPortStr, err := net.SplitHostPort("127.0.0.1:8089")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	tPort, err := strconv.Atoi(tPortStr)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	tests := []struct {
		host string
		port int

		expectedStatus probe.Result
		expectedError  error
	}{
		// A connection is made and probing would succeed
		{tHost, tPort, probe.Success, nil},
		// No connection can be made and probing would fail
		{tHost, -1, probe.Failure, nil},
	}

	prober := New()
	for i, tt := range tests {
		status, _, err := prober.Probe(tt.host, tt.port, 1*time.Second)
		if status != tt.expectedStatus {
			t.Errorf("#%d: expected status=%v, get=%v", i, tt.expectedStatus, status)
		}
		if err != tt.expectedError {
			t.Errorf("#%d: expected error=%v, get=%v", i, tt.expectedError, err)
		}
	}
}
