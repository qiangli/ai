package shell

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// pager divides and prints the given output "page by page" using the size of the
// current terminal.
//
// Navigation keys:
//
//		n Space/Enter : forward one page
//		b             : back one page
//		q             : quit / cancel
//		<number>      : jump to that page number (1-based)
//	 Arrow keys       : navigate using up/down
func pager(output string) error {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24
	}
	_ = width

	lines := strings.Split(output, "\n")
	totalLines := len(lines)

	pageSize := height - 1
	if pageSize < 1 {
		pageSize = 1
	}
	totalPages := (totalLines + pageSize - 1) / pageSize

	// NOTE: return?
	if totalPages <= 1 {
		fmt.Print(output)
		return nil
	}

	fd := int(os.Stdin.Fd())

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(fd, oldState) }() // Best effort.

	// // set to blocking mode to read input
	// // some util may set stdin to non-blocking mode, e.g cat
	// // Get original flags
	// origFlags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	// if err != nil {
	// 	return err
	// }
	// defer func() {
	// 	// Restore original flags
	// 	_, _ = unix.FcntlInt(uintptr(fd), unix.F_SETFL, origFlags)
	// }()
	// // Clear O_NONBLOCK flag
	// newFlags := origFlags &^ unix.O_NONBLOCK
	// // Set flags back (blocking mode)
	// _, err = unix.FcntlInt(uintptr(fd), unix.F_SETFL, newFlags)
	// if err != nil {
	// 	return err
	// }

	reader := bufio.NewReader(os.Stdin)
	current := 0
	numBuffer := ""

	SPACE := byte(0x20)
	CTRL_C := byte(0x03)
	CTRL_D := byte(0x04)

	for {
		fmt.Print("\033[H\033[2J")
		start := current * pageSize
		end := start + pageSize
		if end > totalLines {
			end = totalLines
		}
		for _, line := range lines[start:end] {
			fmt.Printf("\033[1G%s\n", line)
		}
		fmt.Printf("\033[1G-- %d/%d -- b: back, n: next, q: quit #num > ", current+1, totalPages)

		input, err := reader.ReadByte()
		if err != nil {
			fmt.Println("\nError reading input.")
			break
		}

		switch input {
		case 'q', 'Q', CTRL_C, CTRL_D:
			fmt.Printf("\n\033[1G\n")
			return nil
		case 'b', 'B':
			if current > 0 {
				current--
			}
		case '\r', '\n', SPACE, 'n', 'N':
			if current < totalPages-1 {
				current++
			} else {
				current = totalPages - 1
			}
		case 0x1b:
			next, _ := reader.Peek(2)
			if len(next) >= 2 && next[0] == '[' {
				switch next[1] {
				case 'A':
					if current > 0 {
						current--
					}
				case 'B':
					if current < totalPages-1 {
						current++
					}
				case 'C':
					if current < totalPages-1 {
						current++
					}
				case 'D':
					if current > 0 {
						current--
					}
				}
				reader.ReadByte()
				reader.ReadByte()
			}
		default:
			if input >= '0' && input <= '9' {
				numBuffer += string(input)
				gotoPage, _ := strconv.Atoi(numBuffer)
				if gotoPage >= 1 && gotoPage <= totalPages {
					current = gotoPage - 1
				}
			} else {
				numBuffer = ""
			}
		}
	}
	return nil
}
