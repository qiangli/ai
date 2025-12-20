package conf

import (
	"io"
	"os"
	"testing"
)

func TestLoadToolData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	file, err := os.Open("testdata/faas.yaml")
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	d, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	ac, err := LoadToolData([][]byte{d})
	if err != nil {
		t.Fatalf("failed to load file: %v", err)
	}
	for _, v := range ac.Tools {
		t.Logf("%s", v.Body.Script)
	}
}
