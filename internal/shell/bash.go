package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

const usage = `AI Command Line Tool

Usage:
  [ai] message...
  [ai] /[binary] [message...]
  [ai] @[agent] [message...]

Use "%s help" for more info.
`

func Bash(cfg *internal.AppConfig) error {
	var name string
	var args []string

	if len(cfg.Args) > 0 {
		name = cfg.Args[0]
		args = cfg.Args[1:]
	}

	// default to shell
	if name == "" {
		name = "bash"
		if s := os.Getenv("SHELL"); s != "" {
			name = s
		}
	}
	bin, err := exec.LookPath(name)
	if err != nil {
		return err
	}

	// log.Printf(usage, cfg.CommandPath)

	c := exec.Command(bin, args...)
	c.Env = os.Environ()
	// Prompt string for the shell
	// c.Env = append(c.Env, "PS1=ai> ")
	c.Dir = cfg.WorkDir

	// Start the command with a pty.
	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Infof("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// // Copy stdin to the pty and the pty to stdout.
	// // NOTE: The goroutine will keep reading until the next keystroke before returning.
	// go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	// _, _ = io.Copy(os.Stdout, ptmx)

	// Create log files.
	inputLog, err := os.Create("/tmp/input.log")
	if err != nil {
		return err
	}
	defer inputLog.Close()

	outputLog, err := os.Create("/tmp/output.log")
	if err != nil {
		return err
	}
	defer outputLog.Close()

	cleanInputLog := NewStripAnsiWriter(inputLog)
	cleanOutputLog := NewStripAnsiWriter(outputLog)

	// Use a buffered reader to read input line-by-line
	stdinReader := bufio.NewReader(os.Stdin)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdinReader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintln(os.Stderr, "Error reading input:", err)
				continue
			}

			// Processing each input character or sequence
			input := string(buf[:n])
			// Check if input contains newline or specific command
			// if strings.TrimSpace(input) == "ai" {
			// if strings.Contains(input, "ai") {
			// 	// Substitute with your desired command or handle
			// 	input = "your_new_command\n"
			// }

			// Write to ptmx and log
			_, _ = cleanInputLog.Write([]byte(input))
			_, _ = ptmx.Write([]byte(input))
		}
	}()

	// Copy pty output to stdout and log to outputLog.
	ptyReader := io.TeeReader(ptmx, cleanOutputLog)
	// _, _ = io.Copy(os.Stdout, ptyReader)

	// Channel to communicate between goroutines
	outChan := make(chan string)

	go Tail(outChan)

	go func() {
		scanner := bufio.NewScanner(ptyReader)
		for scanner.Scan() {
			line := scanner.Text()
			outChan <- line
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading pty:", err)
		}
	}()

	_, _ = io.Copy(os.Stdout, ptyReader)

	return nil
}

// func stripAnsiControlCharacters(input string) string {
// 	return strings.Map(func(r rune) rune {
// 		// Typical ranges for ASCII control codes
// 		if r < 32 || r == 127 {
// 			return -1
// 		}
// 		return r
// 	}, input)
// }
