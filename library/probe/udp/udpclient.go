package udp

import (
	"fmt"
	"k8s.io/klog"
	"net"
	"strconv"
	"time"
)

func scanUDP(host string, port int, testData string) (err error) {
	address := host + ":" + strconv.Itoa(port)
	serverAddr, err := net.ResolveUDPAddr("udp", address)
	klog.Infoln("Scan UDP Endpoint: ", address)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Write 3 times to the udp socket and check
	// if there's any kind of error
	_ = conn.SetWriteDeadline(time.Now().Add(time.Duration(1) * time.Second))

	if testData == "" {
		//testData = "1234567890"
		testData = "6d6b0100000100000000000002717103636f6d0000010001"
	}
	// query qq hex "6d6b0100000100000000000002717103636f6d0000010001"
	errorCount := 0
	for i := 0; i < 5; i++ {
		buf := []byte(testData + "\n")
		_, err := conn.Write(buf)
		if err != nil {
			errorCount++
		}
	}
	if errorCount > 0 {
		return err
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Duration(1) * time.Second))
	buf := make([]byte, 1024)
	read, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if read > 0 {
		fmt.Println(string(buf[:read]))
		return nil
	} else {
		return err
	}
}

func main() {

	err := scanUDP("114.114.114.114", 53, "")
	if err != nil {
		fmt.Println("192.168.3.78:9999 端口未开")
	} else {

		fmt.Println("192.168.3.78:9999端口开放")
	}

	err1 := scanUDP("192.168.3.78", 8888, "ABCD")
	if err1 != nil {
		fmt.Println("192.168.3.78:8888端口未开")
	} else {
		fmt.Println("192.168.3.78:8888端口开放")
	}

}
