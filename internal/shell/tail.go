package shell

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	captureBeginPattern = ".*#\\[\\[\\s*$"
	captureEndPattern   = ".*#\\]\\]\\s*$"
	aiRunPattern        = ".*#ai\\s*(.*?)\\s*$"
)

func runCommand(command string) {
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("%s\n", output)
	}
}

func Tail(lineChan <-chan string) error {
	beginRegex := regexp.MustCompile(captureBeginPattern)
	endRegex := regexp.MustCompile(captureEndPattern)
	runRegex := regexp.MustCompile(aiRunPattern)

	var lines []string
	var captured []string
	var capturing = false

	process := func(line string) {
		// reset
		if beginRegex.MatchString(line) {
			captured = []string{}
			capturing = true
			return
		}

		var captureEnd bool
		var aiRun bool
		var aiCommand string

		if endRegex.MatchString(line) {
			captureEnd = true
		}
		if runRegex.MatchString(line) {
			aiRun = true
			matches := runRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				aiCommand = matches[1]
			}
		}

		ending := (captureEnd || aiRun)

		if capturing && !ending {
			captured = append(captured, line)
			return
		}

		if ending {
			capturing = false
			lines = append(lines, strings.Join(captured, "\n"))
			captured = []string{}
		}

		if captureEnd {
			return
		}

		if aiRun {
			runCommand(aiCommand)
			return
		}
	}

	for line := range lineChan {
		process(line)
	}

	return nil
}
