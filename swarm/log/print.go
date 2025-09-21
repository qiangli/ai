package log

import (
	"fmt"
	"io"
	"os"
)

// var printLogger Printer = NewPrinter(os.Stdout, false, 0)

// var debugLogger Printer = NewPrinter(os.Stderr, false, 500)

// var infoLogger Printer = NewPrinter(os.Stderr, false, 0)
// var errLogger Printer = NewPrinter(os.Stderr, false, 0)
// var promptLogger Printer = NewPrinter(os.Stderr, false, 0)

type defaultLogger struct {
	logLevel Level

	printLogger  Printer
	debugLogger  Printer
	infoLogger   Printer
	errLogger    Printer
	promptLogger Printer
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		logLevel:     Normal,
		printLogger:  NewPrinter(os.Stdout, false, 0),
		debugLogger:  NewPrinter(os.Stderr, false, 500),
		infoLogger:   NewPrinter(os.Stderr, false, 0),
		errLogger:    NewPrinter(os.Stderr, false, 0),
		promptLogger: NewPrinter(os.Stderr, false, 0),
	}
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
	// Print(...any)
	// Println(...any)

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

// // prompter
// func Promptf(format string, a ...interface{}) {
// 	promptLogger.Printf(format, a...)
// }

// func Prompt(a ...interface{}) {
// 	promptLogger.Print(a...)
// }

// func Promptln(a ...interface{}) {
// 	promptLogger.Println(a...)
// }

// Printer for standard output - for data only (quiet mode)
// func Printf(format string, a ...interface{}) {
// 	printLogger.Printf(format, a...)
// }

// func Print(a ...interface{}) {
// 	printLogger.Print(a...)
// }

// func Println(a ...interface{}) {
// 	printLogger.Println(a...)
// }

// Debug logger
// func Debugf(format string, a ...interface{}) {
// 	debugLogger.Printf(format, a...)
// }

// func Debug(a ...interface{}) {
// 	debugLogger.Print(a...)
// }

// func Debugln(a ...interface{}) {
// 	debugLogger.Println(a...)
// }

// Info logger
// func Infof(format string, a ...interface{}) {
// 	infoLogger.Printf(format, a...)
// }

// func Info(a ...interface{}) {
// 	infoLogger.Print(a...)
// }

// func Infoln(a ...interface{}) {
// 	infoLogger.Println(a...)
// }

// Error logger
// func Errorf(format string, a ...interface{}) {
// 	errLogger.Printf(format, a...)
// }

// func Error(a ...interface{}) {
// 	errLogger.Print(a...)
// }

// func Errorln(a ...interface{}) {
// 	errLogger.Println(a...)
// }

type Level int

const (
	Quiet Level = iota
	Normal
	Verbose
	Tracing
)

// var logLevel Level

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

// Set log level to Normal by default
// func init() {
// 	SetLogLevel(Normal)
// }

func (r *defaultLogger) IsTrace() bool {
	return r.logLevel == Tracing
}
