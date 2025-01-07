package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EmbeddingFunc func(string) ([]float32, error)

type Vector struct {
	ID       int
	UID      string
	Created  string
	Updated  string
	Accessed string
	Content  string

	Score float32
}

type VectorStore struct {
	idx *Index
	ds  *DataStore

	dsn string
	ann string

	reindex   bool
	dimension int

	mu sync.Mutex
}

func New(f int, dsn string) (*VectorStore, error) {
	idx := NewIndex(f)
	ds, err := NewDB(dsn)
	if err != nil {
		return nil, err
	}

	ddl := `CREATE TABLE IF NOT EXISTS vector (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"uid" TEXT,
		"created" timestamp,
		"updated" timestamp,
		"accessed" timestamp,
		"content" TEXT		
	  );`

	if _, err := ds.CreateTable(ddl); err != nil {
		return nil, err
	}

	ann := filepath.Join(filepath.Dir(dsn), filepath.Base(dsn)+".ann")
	exists, err := fileExists(ann)
	if err != nil {
		return nil, err
	}
	if exists {
		if err := idx.Load(ann); err != nil {
			return nil, err
		}
		return &VectorStore{idx: idx, ds: ds, dsn: dsn, ann: ann, reindex: true, dimension: f}, nil
	}
	return &VectorStore{idx: idx, ds: ds, dsn: dsn, ann: ann, reindex: false, dimension: f}, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (vs *VectorStore) getByUID(uid string) (*Vector, error) {
	row := vs.ds.db.QueryRow(`
		SELECT * FROM vector
		WHERE uid = ?
		ORDER BY id DESC
		LIMIT 1
	`, uid)

	var v Vector
	if err := row.Scan(&v.ID, &v.UID, &v.Created, &v.Updated, &v.Accessed, &v.Content); err != nil {
		return nil, err
	}

	return &v, nil
}

func (vs *VectorStore) insert(content string) (string, error) {
	uid := genUID()
	ts := currentTS()
	_, err := vs.ds.Execute(`
		INSERT INTO vector (uid, created, updated, accessed, content)
		VALUES (?, ?, ?, ?, ?)
	`, uid, ts, ts, ts, content)
	if err != nil {
		return "", err
	}

	return uid, nil
}

func (vs *VectorStore) getByID(id uint32) (*Vector, error) {
	row := vs.ds.db.QueryRow(`
		SELECT * FROM vector
		WHERE id = ?
		ORDER BY id DESC
		LIMIT 1
	`, id)

	var v Vector
	if err := row.Scan(&v.ID, &v.UID, &v.Created, &v.Updated, &v.Accessed, &v.Content); err != nil {
		return nil, err
	}

	return &v, nil
}

// Search for similar content in the vector store
func (vs *VectorStore) Search(query string, n int, fn EmbeddingFunc) ([]*Vector, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.reindex {
		vs.idx.Build(10)
	}

	embedding, err := fn(query)
	if err != nil {
		return nil, err
	}
	ids, scores := vs.idx.GetByVector(embedding, n)
	var vectors []*Vector
	for i, id := range ids {
		vector, err := vs.getByID(uint32(id))
		if err != nil {
			return nil, err
		}
		vector.Score = scores[i]
		vectors = append(vectors, vector)
	}
	return vectors, nil
}

// Add content to the vector store
func (vs *VectorStore) Add(content string, fn EmbeddingFunc) error {
	// reindex the index if necessary
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.reindex {
		oldIdx := vs.idx
		newIdx := NewIndex(vs.dimension)
		// loop through all the vectors in the datastore
		rows, err := vs.ds.db.Query("SELECT * FROM vector")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v Vector
			if err := rows.Scan(&v.ID, &v.UID, &v.Created, &v.Updated, &v.Accessed, &v.Content); err != nil {
				return err
			}
			id := uint32(v.ID)
			embedding := oldIdx.GetItem(id)
			if len(embedding) == vs.dimension {
				newIdx.AddItem(id, embedding)
			}
		}
		vs.idx = newIdx
		vs.reindex = false
	}

	uid, err := vs.insert(content)
	if err != nil {
		return err
	}

	// update the index
	v, err := vs.getByUID(uid)
	if err != nil {
		return err
	}

	// get the embedding
	embedding, err := fn(v.Content)
	if err != nil {
		return err
	}
	id := uint32(v.ID)
	vs.idx.AddItem(id, embedding)
	return nil
}

func (vs *VectorStore) Close() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	if !vs.reindex {
		vs.idx.Build(10)
	}
	errs := []error{}
	if err := vs.idx.Save(vs.ann); err != nil {
		errs = append(errs, err)
	}
	if err := vs.idx.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := vs.ds.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("error saving index %+v", errs)
	}
	return nil
}

func genUID() string {
	return uuid.New().String()
}

func currentTS() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
