package util

import (
	"fmt"
	"net"
)

func FindFreePort(startPort int) int {
	port := startPort
	iterations := 0
	for {
		l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err == nil {
			l.Close()
			return port
		}
		port++
		iterations++
		if iterations > 1000 {
			// Let's not go too crazy
			return -1
		}
	}
}
