package db

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestVectorStore(t *testing.T) {
	f := 1_536 // Length of item vector that will be indexed
	n := 3000  // Number of items

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.db")

	vs, err := New(f, filename)
	if err != nil {
		t.Fatalf("Failed to create vector store: %v", err)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	embedding := func(content string) ([]float32, error) {
		v := make([]float32, f)
		for j := range v {
			v[j] = rnd.Float32()
		}
		return v, nil
	}

	for i := 0; i < n; i++ {
		err := vs.Add(fmt.Sprintf("content: %v", i), embedding)
		if err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
	}

	vs.Close()
	t.Log("Vector store saved")

	vs2, err := New(f, filename)
	if err != nil {
		t.Fatalf("Failed to create vector store: %v", err)
	}
	// FIXME: blocking
	// defer vs2.Close()

	result, err := vs2.Search("content: 0", n, embedding)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	t.Logf("ids: %v", len(result))
}

func TestVectorStoreSearch(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.db")

	f := 5

	vs, err := New(f, filename)
	if err != nil {
		t.Fatalf("Failed to create vector store: %v", err)
	}

	//
	embeddings := map[string][]float32{
		"1": {1, 2, 3, 0, 0},
		"2": {4, 5, 6, 0, 0},
		"3": {7, 8, 9, 0, 0},
		"4": {2, 3, 4, 0, 0},
	}
	fake := func(content string) ([]float32, error) {
		return embeddings[content], nil
	}
	query := "1"
	expected := []string{"1", "4", "2", "3"}
	n := 4

	// add items
	for content, _ := range embeddings {
		err := vs.Add(content, fake)
		if err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
	}
	// vs.Build(10)
	err = vs.Close()
	if err != nil {
		t.Fatalf("Failed to close vector store: %v", err)
	}

	// search
	vs2, err := New(f, filename)
	if err != nil {
		t.Fatalf("Failed to create vector store: %v", err)
	}
	// FIXME: blocking
	// defer vs2.Close()

	results, err := vs2.Search(query, n, fake)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	got := make([]string, len(results))
	for i, v := range results {
		got[i] = v.Content
	}
	// verify
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("got: %v, want: %v", got, expected)
	}

	// add more items
	for content, _ := range embeddings {
		err := vs2.Add(content, fake)
		if err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
	}
	expected = []string{"1", "1", "4", "4", "2", "2", "3", "3"}
	n = 8
	// vs2.Build(10)

	results, err = vs2.Search(query, n, fake)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	t.Logf("vector: %+v", got)

	got = make([]string, len(results))
	for i, v := range results {
		got[i] = v.Content
	}
	// verify
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("got: %v, want: %v", got, expected)
	}
	t.Logf("vector: %+v", got)
}
