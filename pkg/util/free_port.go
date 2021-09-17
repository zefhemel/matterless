package util

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

var handedOutPorts = map[int]bool{}

func FindFreePort(startPort int) int {
	rand.Seed(time.Now().UnixNano())
	port := startPort + rand.Intn(10000)

	iterations := 0
	for {
		if !handedOutPorts[port] {
			l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
			if err == nil {
				l.Close()
				handedOutPorts[port] = true
				return port
			}
		}
		port = startPort + rand.Intn(10000)
		iterations++
		if iterations > 1000 {
			// Let's not go too crazy
			return -1
		}
	}
}
