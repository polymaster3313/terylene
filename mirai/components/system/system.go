package system

import (
	"net"
	"runtime"
)

// get ip through google dns
func GETIPDNS() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

// get system information
func GETSYSTEM() (string, string) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	return os, arch
}

// get Wifi local ip through iterface
func GetIpInterface() string {
	var result string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Name != "Wi-Fi" && iface.Name != "wlan0" {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return ""
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				result = ipnet.IP.String()
			}
		}
	}
	return result
}
