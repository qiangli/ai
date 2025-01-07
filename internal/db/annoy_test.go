package db

import (
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestGetItem(t *testing.T) {
	f := 512   // Length of item vector that will be indexed
	n := 10000 // Number of items

	vs := NewIndex(f)

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < n; i++ {
		v := make([]float32, f)
		for j := range v {
			v[j] = rnd.Float32()
		}
		vs.AddItem(uint32(i), v)
	}

	vs.Build(10)

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.ann")

	err := vs.Save(filename)
	if err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	vs2 := NewIndex(f)
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
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	// tmpDir := t.TempDir()
	// filename := filepath.Join(tmpDir, "test.ann")

	f := 2
	vs := NewIndex(f)
	vectors := [][]float32{
		{1, 2},
		{4, 5},
		{7, 8},
		{0, 0},
	}
	query := []float32{1, 2}
	expected := []uint32{1, 2, 3}
	n := 3

	// f := 3
	// vs := NewIndex(f)
	// vectors := [][]float32{
	// 	{1, 2, 3},
	// 	{4, 5, 6},
	// 	{7, 8, 9},
	// 	{1, 2, 3},
	// 	// {0, 0, 0},
	// }
	// query := []float32{1, 2, 3}
	// expected := []uint32{1, 4, 2, 3}
	// n := 4

	// f := 4
	// vs := NewIndex(f)
	// vectors := [][]float32{
	// 	{1, 2, 3, 0},
	// 	{4, 5, 6, 0},
	// 	{7, 8, 9, 0},
	// 	{1, 2, 3, 0},
	// 	{9, 0, 0, 0},
	// 	// {0, 0, 0, 0},
	// }
	// query := []float32{1, 2, 3, 4}
	// expected := []uint32{1, 4, 2, 3}
	// n := 4

	for i, vector := range vectors {
		vs.AddItem(uint32(i+1), vector)
	}
	vs.Build(10)

	// if err := vs.Save(filename); err != nil {
	// 	t.Fatal(err)
	// }

	// if err := vs.Load(filename); err != nil {
	// 	t.Fatal(err)
	// }

	ids, distances := vs.GetByVector(query, n)
	// verify
	if !reflect.DeepEqual(ids, expected) {
		t.Fatalf("ids: %v, expected: %v", ids, expected)
	}
	t.Logf("ids: %v, distances: %v", ids, distances)
}
