package log

import (
	"fmt"
	"io"
	"os"
)

var printLogger = NewPrinter(os.Stdout)

var promptLogger = NewPrinter(os.Stderr)
var debugLogger Printer = NewPrinter(os.Stderr)
var infoLogger = NewPrinter(os.Stderr)
var errLogger = NewPrinter(os.Stderr)

type Printer interface {
	Printf(string, ...interface{})
	Print(...interface{})
	Println(...interface{})

	SetEnabled(bool)
	IsEnabled() bool

	SetLogger(io.Writer)
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

	logger io.Writer
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
	if r.logger != nil {
		fmt.Fprintf(r.logger, format, a...)
	}
}

func (r *printer) Print(a ...interface{}) {
	if r.on {
		fmt.Fprint(r.out, a...)
	}
	if r.logger != nil {
		fmt.Fprint(r.logger, a...)
	}
}

func (r *printer) Println(a ...interface{}) {
	if r.on {
		fmt.Fprintln(r.out, a...)
	}
	if r.logger != nil {
		fmt.Fprintln(r.logger, a...)
	}
}

func (r *printer) SetLogger(w io.Writer) {
	r.logger = w
}

// prompter
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
	Quiet Level = iota
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
	promptLogger.SetEnabled(false)

	switch level {
	case Quiet:
		return
	case Normal:
		infoLogger.SetEnabled(true)
		errLogger.SetEnabled(true)
		promptLogger.SetEnabled(true)
	case Verbose:
		debugLogger.SetEnabled(true)
		infoLogger.SetEnabled(true)
		errLogger.SetEnabled(true)
		promptLogger.SetEnabled(true)
	}

	// Check if stdin is piped/redirected
	if in, err := os.Stdin.Stat(); err == nil {
		// piped | or redirected <
		if in.Mode()&os.ModeNamedPipe != 0 || in.Size() > 0 {
			promptLogger.SetEnabled(false)
		}
	}
}

func SetLogOutput(w io.Writer) {
	printLogger.SetLogger(w)
	debugLogger.SetLogger(w)
	infoLogger.SetLogger(w)
	errLogger.SetLogger(w)
	promptLogger.SetLogger(w)
}

func init() {
	SetLogLevel(Normal)
}
