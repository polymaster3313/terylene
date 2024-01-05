package setup

import (
	"fmt"
	"os"
	"os/exec"
	"terylene/config"
	"time"

	poly "terylene/server/theme/default"

	"github.com/fatih/color"
	zmq "github.com/pebbe/zmq4"
)

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func Setup() (terminal *color.Color, nrouter, npublisher *zmq.Socket) {

	terminalcolor := color.New(color.FgCyan).Add(color.BgHiBlack)

	clearScreen()

	color.Cyan(poly.Title)

	time.Sleep(50 * time.Millisecond)

	publisher, err := zmq.NewSocket(zmq.PUB)

	if err != nil {
		fmt.Println(err)
		color.Red("[X]error creating zeroC2 broadcast")
		os.Exit(1)
	}

	err = publisher.Bind(fmt.Sprintf("tcp://%s:%s", config.C2ip, config.Broadcastport))

	if err != nil {
		fmt.Println(err)
		color.Red("[X] error binding port %s", config.Broadcastport)
		os.Exit(1)
	}

	color.Green("[✔] successfully started zeroC2 broadcast")

	router, err := zmq.NewSocket(zmq.ROUTER)

	if err != nil {
		color.Red("[X] error creating router socket")
	}

	err = router.Bind(fmt.Sprintf("tcp://%s:%s", config.C2ip, config.Routerport))

	router.SetLinger(0)
	if err != nil {
		fmt.Println(err)
		color.Red("[X] error binding port %s", config.Routerport)
		os.Exit(1)
	}

	color.Green("[✔] successfully started zeroC2 router")

	return terminalcolor, router, publisher

}
