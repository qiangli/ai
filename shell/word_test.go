package shell

import (
	"strings"
	"testing"
)

func TestAddWord(t *testing.T) {
	lines := []string{
		"ls /usr/bin",
		"echo hello world",
		"cd /home/user/dev",
		"vi /etc/hosts",
		"cd /home/user/dev",
		"cat /home/user/.bashrc",
		"world world world",
		"cd /home/user/dev",
		"5aa60da1-98c4-49ac-bd2d-809cd64b543c",
		"echo /etc/passwd world",
		"cd /tmp",
		"cd /home/user/dev",
		"world",
		"cd /tmp",
	}

	var counter = NewWordCounter()

	Capture := func(which int, line string) error {
		words := strings.Fields(line)
		counter.AddWords(words)
		return nil
	}

	TopN := func(n int) []WordFreq {
		return counter.TopN(n)
	}

	for _, line := range lines {
		err := Capture(1, line)
		if err != nil {
			t.Errorf("Capture error: %v", err)
		}
	}

	top := TopN(5)
	for _, wf := range top {
		t.Logf("%-18s (freq: %-2d, score: %d)\n", wf.Word, wf.Count, wf.Score)
	}
}
