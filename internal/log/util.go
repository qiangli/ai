package log

import (
	"fmt"
	"io"
)

// Fprintf formats according to a format specifier and writes to w. It truncates the output to max characters.
// if max is 0 or trace is on, it will not truncate.
func Fprintf(w io.Writer, format string, max int, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	s = clip(s, max)
	fmt.Fprint(w, s)
}

func Fprintln(w io.Writer, max int, a ...interface{}) {
	s := fmt.Sprintln(a...)
	s = clip(s, max)
	fmt.Fprint(w, s)
}

func Fprint(w io.Writer, max int, a ...interface{}) {
	s := fmt.Sprint(a...)
	s = clip(s, max)
	fmt.Fprint(w, s)
}

func clip(s string, max int) string {
	if !Trace && max > 0 && len(s) > max {
		trailing := "..."
		if s[len(s)-1] == '\n' || s[len(s)-1] == '\r' {
			trailing = "...\n"
		}
		s = s[:max] + trailing
	}
	return s
}
