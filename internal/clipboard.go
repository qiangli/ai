package internal

import (
	"time"

	"github.com/atotto/clipboard"
)

func WriteToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func ClearClipboard() error {
	return clipboard.WriteAll("")
}

// ReadFromClipboard reads from clipboard
// it will wait until clipboard has content
func ReadFromClipboard() (string, error) {
	for {
		content, err := clipboard.ReadAll()
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}
