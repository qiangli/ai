package watch

import "testing"

func TestWatcherNew(t *testing.T) {
	// structural test
	w := New(nil, time.Second)
	if w.debounce != time.Second {
		t.Error("debounce not set")
	}
}