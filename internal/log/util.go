package log

import (
	"fmt"
	"io"
)

// Fprintf formats according to a format specifier and writes to w. It truncates the output to max characters.
// if max is 0 or trace is on, it will not truncate.
func Fprintf(w io.Writer, format string, max int, a ...interface{}) {
	s := fmt.Sprintf(format, a...)

	if !Trace && max > 0 && len(s) > max {
		s = s[:max] + "..."
	}
	fmt.Fprint(w, s)
}

func Fprintln(w io.Writer, max int, a ...interface{}) {
	Fprintf(w, "%v\n", max, a...)
}

func Fprint(w io.Writer, max int, a ...interface{}) {
	Fprintf(w, "%v", max, a...)
}
