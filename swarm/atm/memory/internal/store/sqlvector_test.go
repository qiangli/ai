package store

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"modernc.org/sqlite"
)

func TestSqlVector(t *testing.T) {
	db := sql.OpenDB(sqlite.Open("file::memory:"))
	defer db.Close()
	CreateTable(db)

	rand.Seed(time.Now().UnixNano())
	emb1 := make([]float64, dim)
	for i := range emb1 {
		emb1[i] = rand.Float64()
	}
	InsertVector(db, "c1", "test.md", 1, 10, "content", emb1)

	// check pack unpack
	rows, _ := db.Query("SELECT embedding FROM vectors WHERE id='c1'")
	var embB []byte
	rows.Scan(&embB)
	rows.Close()
	unpacked, err := UnpackEmbedding(embB)
	if err != nil {
		t.Fatal(err)
	}
	if len(unpacked) != dim {
		t.Fatal("unpack wrong")
	}
}
