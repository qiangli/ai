package db

import (
	"testing"
)

func TestKVStore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tmp := t.TempDir()
	kvs := NewKVStore(tmp)

	if err := kvs.Open("kv.db"); err != nil {
		t.Fatal(err)
	}
	defer kvs.Close()

	const bucket = "chats"

	if err := kvs.Create(bucket); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		key string
		val string
	}{
		{"hello", "world"},
		{"key", "val"},
	}

	// add
	for _, tt := range tests {
		err := kvs.Put(bucket, tt.key, tt.val)
		if err != nil {
			t.Fatal(err)
		}
	}

	// get
	for _, tt := range tests {
		val, err := kvs.Get(bucket, tt.key)
		if err != nil {
			t.Fatal(err)
		}
		if val == nil || *val != tt.val {
			t.Fatal(err)
		}
	}

	// remove
	for _, tt := range tests {
		err := kvs.Delete(bucket, tt.key)
		if err != nil {
			t.Fatal(err)
		}
	}

	// not found exptected
	for _, tt := range tests {
		val, err := kvs.Get(bucket, tt.key)
		if val != nil {
			t.Fatalf("val is not expected %s", *val)
		}
		if err != nil {
			t.Fatalf("unexpected error message: %s", err.Error())
		}
	}

	if err := kvs.Drop(bucket); err != nil {
		t.Fatal(err)
	}
}
