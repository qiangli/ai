package store

import (
	"database/sql"
	"encoding/binary"
	"math"
	"sort"

	"modernc.org/sqlite"
)

const dim = 768

func CreateTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS vectors (
		id TEXT PRIMARY KEY,
		path TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		content TEXT NOT NULL,
		embedding BLOB
	)`)
	return err
}

func PackEmbedding(emb []float64) []byte {
	b := make([]byte, dim*8)
	for i, v := range emb {
		binary.LittleEndian.PutFloat64(b[i*8:], v)
	}
	return b
}

func UnpackEmbedding(b []byte) ([]float64, error) {
	if len(b) != dim*8 {
		return nil, fmt.Errorf("invalid size %d", len(b))
	}
	res := make([]float64, dim)
	for i := 0; i < dim; i++ {
		res[i] = binary.LittleEndian.Float64(b[i*8:])
	}
	return res, nil
}

func InsertVector(db *sql.DB, id, path string, startLine, endLine int, content string, emb []float64) error {
	embB := PackEmbedding(emb)
	_, err := db.Exec("INSERT OR REPLACE INTO vectors (id, path, start_line, end_line, content, embedding) VALUES (?, ?, ?, ?, ?, ?)",
		id, path, startLine, endLine, content, embB)
	return err
}

func dot(a, b []float64) float64 {
	minl := len(a)
	if len(b) < minl {
		minl = len(b)
	}
	sum := 0.0
	for i := 0; i < minl; i++ {
		sum += a[i] * b[i]
	}
	return sum
}
