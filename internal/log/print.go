package log

import (
	"fmt"
	"io"
	"os"
)

type Printer interface {
	Printf(string, ...interface{})
	Print(...interface{})
	Println(...interface{})

	SetEnabled(bool)
	IsEnabled() bool
}

func NewPrinter(w io.Writer) Printer {
	return &printer{
		out: w,
		on:  true,
	}
}

type printer struct {
	out io.Writer
	on  bool
}

func (r *printer) SetEnabled(b bool) {
	r.on = b
}

func (r *printer) IsEnabled() bool {
	return r.on
}

func (r *printer) Printf(format string, a ...interface{}) {
	if r.on {
		fmt.Fprintf(r.out, format, a...)
	}
}

func (r *printer) Print(a ...interface{}) {
	if r.on {
		fmt.Fprint(r.out, a...)
	}
}

func (r *printer) Println(a ...interface{}) {
	if r.on {
		fmt.Fprintln(r.out, a...)
	}
}

// prompter
var promptLogger = NewPrinter(os.Stderr)

func SetPromptEnabled(b bool) {
	promptLogger.SetEnabled(b)
}

func Promptf(format string, a ...interface{}) {
	promptLogger.Printf(format, a...)
}

func Prompt(a ...interface{}) {
	promptLogger.Print(a...)
}

func Promptln(a ...interface{}) {
	promptLogger.Println(a...)
}

// Printer for standard output
var printLogger = NewPrinter(os.Stdout)

func Printf(format string, a ...interface{}) {
	printLogger.Printf(format, a...)
}

func Print(a ...interface{}) {
	printLogger.Print(a...)
}

func Println(a ...interface{}) {
	printLogger.Println(a...)
}

// Debug logger
var debugLogger Printer = NewPrinter(os.Stderr)

func SetDebugEnabled(b bool) {
	debugLogger.SetEnabled(b)
}

func Debugf(format string, a ...interface{}) {
	debugLogger.Printf(format, a...)
}

func Debug(a ...interface{}) {
	debugLogger.Print(a...)
}

func Debugln(a ...interface{}) {
	debugLogger.Println(a...)
}

// Info logger
var infoLogger = NewPrinter(os.Stderr)

func Infof(format string, a ...interface{}) {
	infoLogger.Printf(format, a...)
}

func Info(a ...interface{}) {
	infoLogger.Print(a...)
}

func Infoln(a ...interface{}) {
	infoLogger.Println(a...)
}

// Error logger
var errLogger = NewPrinter(os.Stderr)

func Errorf(format string, a ...interface{}) {
	errLogger.Printf(format, a...)
}

func Error(a ...interface{}) {
	errLogger.Print(a...)
}

func Errorln(a ...interface{}) {
	errLogger.Println(a...)
}

type Level int

const (
	Silent Level = iota
	Quiet
	Normal
	Verbose
)

// IsVerbose returns true if debug output is enabled.
func IsVerbose() bool {
	return debugLogger.IsEnabled()
}

func SetLogLevel(level Level) {
	// stdout
	printLogger.SetEnabled(true)

	// stderr
	debugLogger.SetEnabled(false)
	infoLogger.SetEnabled(false)
	errLogger.SetEnabled(false)

	switch level {
	case Silent:
		return
	case Normal:
		infoLogger.SetEnabled(true)
		errLogger.SetEnabled(true)
	case Verbose:
		debugLogger.SetEnabled(true)
		infoLogger.SetEnabled(true)
		errLogger.SetEnabled(true)
	}

	// Check if stdin is piped/redirected
	if in, err := os.Stdin.Stat(); err == nil {
		// piped | or redirected <
		if in.Mode()&os.ModeNamedPipe != 0 || in.Size() > 0 {
			SetPromptEnabled(false)
		}
	}
}

func init() {
	SetLogLevel(Normal)
}
