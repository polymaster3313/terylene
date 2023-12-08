package fade

import (
	"fmt"
	"strings"
	"time"
)

func Amber(text string) string {
	faded := ""
	for _, line := range strings.Split(text, "\n") {
		green := 250
		for _, char := range line {
			green -= 5
			if green < 0 {
				green = 0
			}
			faded += fmt.Sprintf("\033[38;2;255;%d;0m%s\033[0m", green, string(char))
		}
		faded += "\n"
	}
	return faded
}

func Water(text string) string {
	faded := ""
	green := 10
	for _, line := range strings.Split(text, "\n") {
		faded += fmt.Sprintf("\033[38;2;0;%d;255m%s\033[0m\n", green, line)
		if green != 255 {
			green += 15
			if green > 255 {
				green = 255
			}
		}
	}
	return faded
}

func Purple(text string) string {
	faded := ""
	down := false

	for _, line := range strings.Split(text, "\n") {
		red := 40
		for _, char := range line {
			if down {
				red -= 3
			} else {
				red += 3
			}
			if red > 254 {
				red = 255
				down = true
			} else if red < 1 {
				red = 30
				down = false
			}
			faded += fmt.Sprintf("\033[38;2;%d;0;220m%s\033[0m", red, string(char))
		}
	}
	return faded
}

func Rainbow(text string, delay time.Duration) {
	colors := []int{91, 93, 92, 96, 94, 95}

	for _, char := range text {
		color := colors[0]
		colors = append(colors[1:], color)

		fmt.Printf("\033[%dm%s", color, string(char))
		time.Sleep(delay)
	}

	fmt.Print("\033[0m")
}
