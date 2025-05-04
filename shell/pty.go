package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/qiangli/ai/internal/log"
)

func RunPtyCapture(shellBin, command string, capture func(int, string) error) error {
	cmd := exec.Command(shellBin, "-c", command)

	var stdin = os.Stdin
	var stdout = os.Stdout

	// Start the command with a pty.
	ptmx, err := pty.Start(cmd)
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
			if err := pty.InheritSize(stdin, ptmx); err != nil {
				fmt.Printf("error resizing pty: %s\n", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	fd := int(stdin.Fd())
	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(fd, oldState) }() // Best effort.

	var done int32

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	// This is worked around by using a non-blocking read on stdin.
	//
	// keep reading from stdin and write to ptmx until done
	// _, _ = io.Copy(ptmx, stdin)
	in := func() {
		// non blocking read
		// TODO windows
		origFlags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
		if err != nil {
			log.Debugf("stdin fcntl get flags error: %s\n", err)
			return
		}
		defer func() {
			_, _ = unix.FcntlInt(uintptr(fd), unix.F_SETFL, origFlags)
			log.Debugf("stdin restore original flags %+v\n", origFlags)
		}()

		newFlags := origFlags | unix.O_NONBLOCK
		_, err = unix.FcntlInt(uintptr(fd), unix.F_SETFL, newFlags)
		if err != nil {
			log.Debugf("stdin fcntl set flags error: %s\n", err)
			return
		}

		buf := make([]byte, 1024)
		for {
			n, err := stdin.Read(buf)
			if n > 0 {
				ptmx.Write(buf[:n])
			}
			if atomic.LoadInt32(&done) == 1 {
				break
			}
			if err != nil {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	// keep reading from ptmx and write to stdout until EOF
	// set done to 1 when done
	// _, _ = io.Copy(stdout, ptmx)
	out := func() {
		var lineBuf []byte
		buf := make([]byte, 1024)

		flushLine := func() bool {
			if len(lineBuf) > 0 {
				if err := capture(len(lineBuf), string(lineBuf)); err != nil {
					return false
				}
				lineBuf = lineBuf[:0]
			}
			return true
		}

		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				stdout.Write(buf[:n]) // output as is
				for i := 0; i < n; i++ {
					switch buf[i] {
					case '\r':
						flushLine()
						if i+1 < n && buf[i+1] == '\n' {
							i++
						}
					case '\n':
						flushLine()
					default:
						lineBuf = append(lineBuf, buf[i])
					}
				}
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				if err == unix.EAGAIN {
					time.Sleep(50 * time.Millisecond)
					continue
				}
				break
			}
		}
		flushLine() // capture last line if any
		atomic.StoreInt32(&done, 1)
	}

	go in()
	go out()

	cmd.Wait()

	return nil
}
