package util

import (
	"os"
	"testing"
)

func TestPrintImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	imageFile := "/tmp/term.png"
	err := PrintImage(os.Stdout, imageFile)
	if err != nil {
		t.Fatal(err)
	}
}
