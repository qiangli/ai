package model

import (
	"testing"
)

func TestLoadModels(t *testing.T) {
	m, err := LoadModels("testdata/")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m)
}
