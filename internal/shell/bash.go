package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

const usage = `AI Command Line Tool

Usage:
  [ai] message...
  [ai] /[binary] [message...]
  [ai] @[agent] [message...]

Use "%s help" for more info.
`

func Bash(cfg *llm.Config) error {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "bash"
	}
	bin, err := exec.LookPath(shellPath)
	if err != nil {
		return err
	}

	log.Printf(usage, cfg.CommandPath)

	c := exec.Command(bin)
	c.Env = os.Environ()
	c.Env = append(c.Env, "PS1=ai> ")
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
			if strings.Contains(input, "ai") {
				// Substitute with your desired command or handle
				input = "your_new_command\n"
			}

			// Write to ptmx and log
			_, _ = cleanInputLog.Write([]byte(input))
			_, _ = ptmx.Write([]byte(input))
		}
	}()

	// Copy pty output to stdout and log to outputLog.
	ptyReader := io.TeeReader(ptmx, cleanOutputLog)
	_, _ = io.Copy(os.Stdout, ptyReader)

	return nil
}

func stripAnsiControlCharacters(input string) string {
	return strings.Map(func(r rune) rune {
		// Typical ranges for ASCII control codes
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, input)
}
