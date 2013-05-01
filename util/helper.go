// Package util provides various useful functions for completing the
// "Whispering Gophers" code lab.
package util

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
)

// Listen returns a Listener that listens on the first available port on the
// first available non-loopback IPv4 network interface.
func Listen() (net.Listener, error) {
	ip, err := externalIP()
	if err != nil {
		return nil, fmt.Errorf("could not find active non-loopback address: %v", err)
	}
	// Why the ":0"?
	return net.Listen("tcp4", ip+":0")
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

// RandomID returns an 8 byte random string in hexadecimal.
func RandomID() string {
	b := make([]byte, 8)
	n, _ := rand.Read(b)
	return fmt.Sprintf("%x", b[:n])
}
