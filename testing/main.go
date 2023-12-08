package main

import (
	"bufio"
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("history")

	// Get pipe for standard output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating StdoutPipe for Cmd", err)
		return
	}

	// Start command
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
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

			// Print the output
			fmt.Println(string(output))
		}
	}()

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		if err.Error() == "exit status 255" {
			return
		}
		fmt.Println("Error waiting for command:", err)
	}
}
