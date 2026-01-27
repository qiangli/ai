package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
)

type Level = api.LogLevel

const (
	Quiet       Level = api.Quiet
	Informative       = api.Informative
	Verbose           = api.Verbose
	Tracing           = api.Tracing
)

// defaultLogger implements Logger and supports both console and "tee" (file) outputs.
// Console and file outputs have independent enabled levels. The tee file is optional.
type defaultLogger struct {
	logLevel  Level
	fLogLevel Level

	// console printers
	printLogger  Printer
	debugLogger  Printer
	infoLogger   Printer
	errLogger    Printer
	promptLogger Printer

	// file (tee) printers
	fPrintLogger  Printer
	fDebugLogger  Printer
	fInfoLogger   Printer
	fErrLogger    Printer
	fPromptLogger Printer

	// optional file writer used by tee printers
	teeFile *FileWriter
}

func newDefaultLogger() *defaultLogger {
	logger := &defaultLogger{
		logLevel:     Quiet,
		fLogLevel:    Quiet,
		printLogger:  NewPrinter(os.Stdout, false, 0),
		debugLogger:  NewPrinter(os.Stderr, false, 500),
		infoLogger:   NewPrinter(os.Stderr, false, 0),
		errLogger:    NewPrinter(os.Stderr, false, 0),
		promptLogger: NewPrinter(os.Stderr, false, 0),
		// tee printers default to disabled and write to /dev/null
		fPrintLogger:  NewPrinter(io.Discard, false, 0),
		fDebugLogger:  NewPrinter(io.Discard, false, 500),
		fInfoLogger:   NewPrinter(io.Discard, false, 0),
		fErrLogger:    NewPrinter(io.Discard, false, 0),
		fPromptLogger: NewPrinter(io.Discard, false, 0),
	}
	return logger
}

func (r *defaultLogger) Promptf(format string, a ...any) {
	r.promptLogger.Printf(format, a...)
	r.fPromptLogger.Printf(format, a...)
}

func (r *defaultLogger) Printf(format string, a ...any) {
	r.printLogger.Printf(format, a...)
	r.fPrintLogger.Printf(format, a...)
}

func (r *defaultLogger) Errorf(format string, a ...any) {
	r.errLogger.Printf(format, a...)
	r.fErrLogger.Printf(format, a...)
}

func (r *defaultLogger) Infof(format string, a ...any) {
	r.infoLogger.Printf(format, a...)
	r.fInfoLogger.Printf(format, a...)
}

func (r *defaultLogger) Debugf(format string, a ...any) {
	r.debugLogger.Printf(format, a...)
	r.fDebugLogger.Printf(format, a...)
}

// Expose control for tee file. SetTeeFile opens/creates the given pathname and directs tee
// output to it. Pass empty path to disable and close existing tee.
func (r *defaultLogger) SetTeeFile(pathname string) error {
	// close existing
	if r.teeFile != nil {
		_ = r.teeFile.Close()
		r.teeFile = nil
	}
	if pathname == "" {
		// disable tee: set file printers to discard
		r.setFileWriters(io.Discard)
		return nil
	}
	// ensure dir exists
	dir := filepath.Dir(pathname)
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	cw, err := NewFileWriter(pathname)
	if err != nil {
		return err
	}
	r.teeFile = cw
	// set writers for file printers
	r.setFileWriters(cw)
	// enable/disable file printers according to fLogLevel
	r.applyFileLogLevel()
	return nil
}

func (r *defaultLogger) CloseTee() error {
	if r.teeFile == nil {
		return nil
	}
	err := r.teeFile.Close()
	r.teeFile = nil
	// reset file printers to discard and disabled
	r.setFileWriters(io.Discard)
	r.fPrintLogger.SetEnabled(false)
	r.fDebugLogger.SetEnabled(false)
	r.fInfoLogger.SetEnabled(false)
	r.fErrLogger.SetEnabled(false)
	r.fPromptLogger.SetEnabled(false)
	return err
}

func (r *defaultLogger) setFileWriters(w io.Writer) {
	r.fPrintLogger.SetWriter(w)
	r.fDebugLogger.SetWriter(w)
	r.fInfoLogger.SetWriter(w)
	r.fErrLogger.SetWriter(w)
	r.fPromptLogger.SetWriter(w)
}

func (r *defaultLogger) applyFileLogLevel() {
	switch r.fLogLevel {
	case Quiet:
		r.fDebugLogger.SetEnabled(false)
		r.fInfoLogger.SetEnabled(false)
		r.fErrLogger.SetEnabled(false)
		r.fPromptLogger.SetEnabled(false)
	case Informative:
		r.fDebugLogger.SetEnabled(false)
		r.fInfoLogger.SetEnabled(true)
		r.fErrLogger.SetEnabled(true)
		r.fPromptLogger.SetEnabled(true)
	case Verbose, Tracing:
		r.fDebugLogger.SetEnabled(true)
		r.fInfoLogger.SetEnabled(true)
		r.fErrLogger.SetEnabled(true)
		r.fPromptLogger.SetEnabled(true)
	}
}

// SetTeeLogLevel controls the level for file (tee) outputs independently from console.
func (r *defaultLogger) SetTeeLogLevel(level Level) {
	r.fLogLevel = level
	r.applyFileLogLevel()
}

// Printer interface and implementation
type Printer interface {
	Printf(string, ...any)

	SetEnabled(bool)
	IsEnabled() bool

	SetWriter(io.Writer)
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

	// optional separate writer (for tee)
	writer io.Writer
}

func (r *printer) SetEnabled(b bool) {
	r.on = b
}

func (r *printer) IsEnabled() bool {
	return r.on
}

func (r *printer) SetWriter(w io.Writer) {
	r.writer = w
}

func (r *printer) Printf(format string, a ...interface{}) {
	if r.on {
		fmt.Fprintf(r.out, format, a...)
	}
	if r.writer != nil {
		fmt.Fprintf(r.writer, format, a...)
	}
}

func (r *printer) Print(a ...interface{}) {
	if r.on {
		fmt.Fprint(r.out, a...)
	}
	if r.writer != nil {
		fmt.Fprint(r.writer, a...)
	}
}

func (r *printer) Println(a ...interface{}) {
	if r.on {
		fmt.Fprintln(r.out, a...)
	}
	if r.writer != nil {
		fmt.Fprintln(r.writer, a...)
	}
}

func (r *defaultLogger) IsVerbose() bool {
	return r.logLevel == Verbose
}

func (r *defaultLogger) IsQuiet() bool {
	return r.logLevel == Quiet
}

func (r *defaultLogger) IsInformative() bool {
	return r.logLevel == Informative
}

func (r *defaultLogger) IsTrace() bool {
	return r.logLevel == Tracing
}

func (r *defaultLogger) SetLogLevel(level Level) {
	r.logLevel = level

	// stdout
	r.printLogger.SetEnabled(true)

	// stderr levels
	switch level {
	case Quiet:
		r.debugLogger.SetEnabled(false)
		r.infoLogger.SetEnabled(false)
		r.errLogger.SetEnabled(false)
		r.promptLogger.SetEnabled(false)
	case Informative:
		r.debugLogger.SetEnabled(false)
		r.infoLogger.SetEnabled(true)
		r.errLogger.SetEnabled(true)
		r.promptLogger.SetEnabled(true)
	case Verbose, Tracing:
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
