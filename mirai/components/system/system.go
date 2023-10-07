package system

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
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
func getpubIp2() (string, error) {
	cmd := exec.Command("curl", "ifconfig.me")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
func GetpubIp() (string, error) {
	url := "https://api.ipify.org?format=text"
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		ip, err := getpubIp2()
		if err != nil {
			return "", err
		}
		return ip, nil
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(ip), nil
}

func GetOS() (string, error) {
	cmd := exec.Command("cat", "/etc/os-release")

	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", err
	}

	slices := strings.Split(string(output), "\n")
	for _, slice := range slices {
		if strings.Contains(slice, "ID=") && !strings.Contains(slice, "VERSION") && !strings.Contains(slice, "PLATFORM") {
			OS := strings.Split(slice, "=")[1]
			return OS, nil
		}
	}
	return "unidentified OS", nil
}

// Generate a unique connection ID for connection throttling
func GenerateConnID(localIP, pubIP string) string {
	full := localIP + pubIP

	hashBytes := sha256.Sum256([]byte(full))
	first8Bytes := hashBytes[:8]

	hexString := hex.EncodeToString(first8Bytes)

	return hexString
}

// get system information
func GETSYSTEM() (string, string, string) {
	os, err := GetOS()
	if err != nil {
		os = runtime.GOARCH
	}
	arch := runtime.GOARCH

	pubip, err := GetpubIp()
	if err != nil {
		return "", "", ""
	}

	return arch, os, pubip
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
