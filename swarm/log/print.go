package log

import (
	"fmt"
	"io"
	"os"
)

type defaultLogger struct {
	logLevel Level

	printLogger  Printer
	debugLogger  Printer
	infoLogger   Printer
	errLogger    Printer
	promptLogger Printer
}

func newDefaultLogger() *defaultLogger {
	logger := &defaultLogger{
		logLevel:     Normal,
		printLogger:  NewPrinter(os.Stdout, false, 0),
		debugLogger:  NewPrinter(os.Stderr, false, 500),
		infoLogger:   NewPrinter(os.Stderr, false, 0),
		errLogger:    NewPrinter(os.Stderr, false, 0),
		promptLogger: NewPrinter(os.Stderr, false, 0),
	}
	logger.SetLogLevel(Normal)
	return logger
}

func (r *defaultLogger) Prompt(format string, a ...any) {
	r.promptLogger.Printf(format, a...)
}

func (r *defaultLogger) Print(format string, a ...any) {
	r.printLogger.Printf(format, a...)
}

func (r *defaultLogger) Error(format string, a ...any) {
	r.errLogger.Printf(format, a...)
}

func (r *defaultLogger) Info(format string, a ...any) {
	r.infoLogger.Printf(format, a...)
}

func (r *defaultLogger) Debug(format string, a ...any) {
	r.debugLogger.Printf(format, a...)
}

type Printer interface {
	Printf(string, ...any)

	SetEnabled(bool)
	IsEnabled() bool

	SetLogger(io.Writer)
}

func NewPrinter(w io.Writer, enabled bool, max int) Printer {
	return &printer{
		out: w,
		on:  enabled,
		max: max,
	}
}

type printer struct {
	out io.Writer
	on  bool

	max int

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

type Level int

const (
	Quiet Level = iota
	Normal
	Verbose
	Tracing
)

func (r *defaultLogger) IsVerbose() bool {
	return r.logLevel == Verbose
}

func (r *defaultLogger) IsQuiet() bool {
	return r.logLevel == Quiet
}

func (r *defaultLogger) IsNormal() bool {
	return r.logLevel == Normal
}

func (r *defaultLogger) SetLogLevel(level Level) {
	r.logLevel = level

	// stdout
	r.printLogger.SetEnabled(true)

	// stderr
	switch level {
	case Quiet:
		r.debugLogger.SetEnabled(false)
		r.infoLogger.SetEnabled(false)
		r.errLogger.SetEnabled(false)
		r.promptLogger.SetEnabled(false)
	case Normal:
		r.debugLogger.SetEnabled(false)
		r.infoLogger.SetEnabled(true)
		r.errLogger.SetEnabled(true)
		r.promptLogger.SetEnabled(true)
	case Verbose | Tracing:
		r.debugLogger.SetEnabled(true)
		r.infoLogger.SetEnabled(true)
		r.errLogger.SetEnabled(true)
		r.promptLogger.SetEnabled(true)
	}

	// Check if stdin is piped/redirected
	if in, err := os.Stdin.Stat(); err == nil {
		// piped | or redirected <
		if in.Mode()&os.ModeNamedPipe != 0 || in.Size() > 0 {
			r.promptLogger.SetEnabled(false)
		}
	}
}

func (r *defaultLogger) SetLogOutput(w io.Writer) {
	r.printLogger.SetLogger(w)

	r.debugLogger.SetLogger(w)
	r.infoLogger.SetLogger(w)
	r.errLogger.SetLogger(w)
	r.promptLogger.SetLogger(w)
}

func (r *defaultLogger) IsTrace() bool {
	return r.logLevel == Tracing
}
