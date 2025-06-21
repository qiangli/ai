package util

import (
	"testing"
)

func TestNotify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	Notify("Ready")
}

func TestAlert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	Alert("testing")
}

func TestBeep(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	Beep(3)
}

func TestBell(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	Bell(3)
}
