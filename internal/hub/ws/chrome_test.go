package ws

import (
	"testing"
)

func TestGetSelection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	txt, err := GetSelection("ws://localhost:58080/hub")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got: %s", txt)
}

func TestVoiceInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	txt, err := VoiceInput("ws://localhost:58080/hub", "Speak!")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got: %s", txt)
}

func TestScreenshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	txt, err := Screenshot("ws://localhost:58080/hub", "capturing screenshot")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got: %s", txt)
}
