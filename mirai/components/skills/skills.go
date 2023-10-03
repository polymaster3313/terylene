package skills

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func Pinger(local string, device string) []string {
	var active []string

	parts := strings.Split(local, ".")
	parts = parts[:len(parts)-1]
	baseip := strings.Join(parts, ".") + "."

	var wg sync.WaitGroup
	results := make(chan string, 255)

	sem := make(chan struct{}, 50)

	for i := 1; i < 256; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			targetHost := fmt.Sprintf("%s%d", baseip, i)
			if targetHost == local {
				<-sem
				return
			}
			var cmd *exec.Cmd
			if device == "windows" {
				cmd = exec.Command("ping", "-n", "1", "-w", "1", targetHost)
			} else if device == "linux" {
				cmd = exec.Command("ping", "-c", "1", "-w", "1", targetHost)
			} else {
				<-sem
				return
			}
			output, err := cmd.CombinedOutput()
			if err != nil {
				<-sem
				return
			}
			if !strings.Contains(string(output), "timed out") || !strings.Contains(string(output), "100% packet loss") {
				results <- targetHost
			}
			<-sem
		}(i)
	}

	wg.Wait()
	close(results)

	for result := range results {
		active = append(active, result)
	}
	return active
}

func IsSSHOpen(host string) bool {
	address := fmt.Sprintf("%s:%s", host, "22")
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
