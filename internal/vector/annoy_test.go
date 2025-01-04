package vector

import (
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestGetItem(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	f := 40   // Length of item vector that will be indexed
	n := 1000 // Number of items

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.ann")
	vs := New(f)

	for i := 0; i < n; i++ {
		v := make([]float32, f)
		for j := range v {
			v[j] = rand.Float32()
		}
		vs.AddItem(uint32(i), v)
	}

	vs.Build(10)

	err := vs.Save(filename)
	if err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	vs2 := New(f)
	if err := vs2.Load(filename); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}
	ids, distances := vs2.GetByItem(0, n)
	// Check if the distances are ascending
	for i := 0; i < len(distances)-1; i++ {
		if distances[i] > distances[i+1] {
			t.Fatalf("distances is wrong: %v %v > %v", i, distances[i], distances[i+1])
		}
	}
	t.Logf("ids: %v, dists: %v", ids, distances)
}

func TestGetByVector(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.ann")

	f := 3

	vs := New(f)

	vs.AddItem(1, []float32{1, 2, 3})
	vs.AddItem(2, []float32{4, 5, 6})
	vs.AddItem(3, []float32{7, 8, 9})
	vs.AddItem(4, []float32{2, 3, 4})
	vs.Build(10)

	if err := vs.Save(filename); err != nil {
		t.Fatal(err)
	}

	if err := vs.Load(filename); err != nil {
		t.Fatal(err)
	}

	ids, distances := vs.GetByVector([]float32{1, 2, 3}, 4)
	// verify
	expected := []uint32{1, 4, 2, 3}
	if !reflect.DeepEqual(ids, expected) {
		t.Fatalf("ids: %v, expected: %v", ids, expected)
	}
	t.Logf("ids: %v, distances: %v", ids, distances)
}
