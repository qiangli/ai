package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTeeRespectsLogLevel_DebugSuppressed(t *testing.T) {
	l := newDefaultLogger()
	tdir := t.TempDir()
	path := filepath.Join(tdir, "tee.log")
	if err := l.SetTeeFile(path); err != nil {
		t.Fatalf("SetTeeFile: %v", err)
	}
	// set tee log level to Informative: debug should be disabled
	l.SetTeeLogLevel(Informative)
	l.Debugf("SHOULD_NOT_APPEAR\n")
	// flush/close to ensure writes
	if err := l.CloseTee(); err != nil {
		t.Fatalf("CloseTee: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("expected empty log file, got: %q", string(b))
	}
}

func TestTeeRespectsLogLevel_DebugEmitted(t *testing.T) {
	l := newDefaultLogger()
	tdir := t.TempDir()
	path := filepath.Join(tdir, "tee2.log")
	if err := l.SetTeeFile(path); err != nil {
		t.Fatalf("SetTeeFile: %v", err)
	}
	// set tee log level to Verbose: debug should be enabled
	l.SetTeeLogLevel(Verbose)
	l.Debugf("SHOULD_APPEAR\n")
	if err := l.CloseTee(); err != nil {
		t.Fatalf("CloseTee: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(b), "SHOULD_APPEAR") {
		t.Fatalf("expected debug message in log file, got: %q", string(b))
	}
}
