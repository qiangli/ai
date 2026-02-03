package index

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"atm/memory"
	"atm/memory/internal/chunk"
	"atm/memory/internal/embed"
	"atm/memory/internal/store"
	"atm/memory/internal/watch"

	"github.com/blevesearch/bleve/v2"
)

type MemoryIndex struct {
	index    bleve.Index
	db       *sql.DB
	provider embed.Provider
	cfg      memory.MemoryConfig
	fallback embed.Provider
	watcher  *watch.Watcher
}

func NewMemoryIndex(cfg memory.MemoryConfig) (*MemoryIndex, error) {
	storePath := cfg.StorePath
	if storePath == "" {
		storePath = filepath.Join(os.TempDir(), "memory-store")
	}
	blevePath := filepath.Join(storePath, "bleve")
	_ = os.MkdirAll(blevePath, 0755)
	dbPath := filepath.Join(storePath, "vectors.db")
	pathMapping := bleve.NewTextFieldMapping()
	pathMapping.Analyzed = false
	pathMapping.Indexed = true
	pathMapping.Stored = true
	startLineMapping := bleve.NewNumericFieldMapping()
	startLineMapping.Indexed = true
	startLineMapping.Stored = true
	endLineMapping := bleve.NewNumericFieldMapping()
	endLineMapping.Indexed = true
	endLineMapping.Stored = true
	contentMapping := bleve.NewTextFieldMapping()
	contentMapping.Analyzed = true
	contentMapping.Stored = true
	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappings(pathMapping, startLineMapping, endLineMapping, contentMapping)
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("chunk", docMapping)
	var index bleve.Index
	if exists(filepath.Join(blevePath, "0.index")) {
		var err error
		index, err = bleve.Open(blevePath)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		index, err = bleve.New(blevePath, indexMapping)
		if err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=1&mode=rwc")
	if err != nil {
		return nil, err
	}
	store.CreateTable(db)
	provider := embed.NewProvider(cfg.Provider, cfg.Model)
	var fallback embed.Provider
	if cfg.Fallback != "" {
		fallback = embed.NewProvider(cfg.Fallback, cfg.Model)
	}
	mi := &MemoryIndex{
		index:    index,
		db:       db,
		provider: provider,
		cfg:      cfg,
		fallback: fallback,
	}
	if len(cfg.Sync.Watch) > 0 {
		debounce, _ := time.ParseDuration(cfg.Sync.Debounce)
		w := watch.New(mi, debounce)
		mi.watcher = w
	}
	return mi, nil
}

func (mi *MemoryIndex) Close() error {
	if mi.watcher != nil {
		mi.watcher.Stop()
	}
	mi.db.Close()
	return mi.index.Close()
}

func (mi *MemoryIndex) Search(query string) ([]memory.SearchResult, error) {
	embQ, err := mi.provider.Embed([]string{query})
	fallbackUsed := false
	if err != nil && mi.fallback != nil {
		embQ, err = mi.fallback.Embed([]string{query})
		fallbackUsed = true
	}
	var vecQ []float64
	hasVector := err == nil && len(embQ) > 0
	if hasVector {
		vecQ = embQ[0]
	}
	normQ := math.Sqrt(dot(vecQ, vecQ))
	if normQ == 0 {
		normQ = 1
	}
	candMult := mi.cfg.Query.Hybrid.CandidateMultiplier
	maxCand := candMult * mi.cfg.Query.MaxResults
	textCands := make(map[string]textCand)
	var maxText float64
	{
		textQ := bleve.NewMatchQuery(query)
		sreq := bleve.NewSearchRequest(textQ)
		sreq.Size = maxCand
		sres, err := mi.index.Search(sreq)
		if err != nil {
			log.Printf("bleve search err: %v", err)
		} else {
			for _, hit := range sres.Hits {
				score := hit.Score
				if score > maxText {
					maxText = score
				}
				path, _ := hit.Fields["path"].(string)
				sl, _ := hit.Fields["start_line"].(float64)
				el, _ := hit.Fields["end_line"].(float64)
				cont, _ := hit.Fields["content"].(string)
				id := hit.Id
				textCands[id] = textCand{
					Path:      path,
					StartLine: int(sl),
					EndLine:   int(el),
					Content:   trunc(cont, 700),
					Score:     score,
				}
			}
		}
	}
	vecCands := make(map[string]vecCand)
	var maxVec float64
	if hasVector {
		rows, err := mi.db.Query("SELECT id, path, start_line, end_line, content, embedding FROM vectors")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, path string
				var sl, el int
				var cont string
				var embB []byte
				rows.Scan(&id, &path, &sl, &el, &cont, &embB)
				embD, err := store.UnpackEmbedding(embB)
				if err != nil || len(embD) != dim {
					continue
				}
				normD := math.Sqrt(dot(vecQ, embD))
				if normD == 0 {
					continue
				}
				score := dot(vecQ, embD) / normQ / normD
				if score > maxVec {
					maxVec = score
				}
				vecCands[id] = vecCand{
					Path:      path,
					StartLine: sl,
					EndLine:   el,
					Content:   trunc(cont, 700),
					Score:     score,
				}
			}
		} else {
			log.Printf("vector query err: %v", err)
		}
	}
	// merge
	all := []memory.SearchResult{}
	for id, tc := range textCands {
		normT := tc.Score / maxText
		normV := 0.0
		if vc, ok := vecCands[id]; ok {
			normV = vc.Score / maxVec
		}
		score := mi.cfg.Query.Hybrid.VectorWeight*normV + mi.cfg.Query.Hybrid.TextWeight*normT
		all = append(all, memory.SearchResult{
			Content:      tc.Content,
			Path:         tc.Path,
			StartLine:    tc.StartLine,
			EndLine:      tc.EndLine,
			Score:        score,
			Provider:     mi.provider.Name(),
			Model:        mi.provider.ModelName(),
			FallbackUsed: fallbackUsed,
		})
	}
	for id, vc := range vecCands {
		if _, ok := textCands[id]; ok {
			continue
		}
		normV := vc.Score / maxVec
		score := mi.cfg.Query.Hybrid.VectorWeight*normV
		all = append(all, memory.SearchResult{
			Content:      vc.Content,
			Path:         vc.Path,
			StartLine:    vc.StartLine,
			EndLine:      vc.EndLine,
			Score:        score,
			Provider:     mi.provider.Name(),
			Model:        mi.provider.ModelName(),
			FallbackUsed: fallbackUsed,
		})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	if len(all) > mi.cfg.Query.MaxResults {
		all = all[:mi.cfg.Query.MaxResults]
	}
	return all, nil
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

type textCand struct {
	Path      string
	StartLine int
	EndLine   int
	Content   string
	Score     float64
}

type vecCand textCand

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

func (mi *MemoryIndex) IndexWorkspace(root string) error {
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := filepath.Base(p)
		dirBase := filepath.Base(filepath.Dir(p))
		matched := name == "MEMORY.md" || (dirBase == "memory" && regexp.MustCompile(`^\\d{4}-\\d{2}-\\d{2}\\.md$`).MatchString(name))
		if !matched {
			for _, pat := range mi.cfg.ExtraPaths {
				if m, _ := filepath.Match(pat, p); m {
					matched = true
					break
				}
			}
		}
		if matched {
			return mi.IndexFile(p)
		}
		return nil
	})
	return err
}

func (mi *MemoryIndex) IndexFile(path string) error {
	pathQuery := bleve.NewTermQuery(path).SetField("path")
	sreq := bleve.NewSearchRequest(pathQuery)
	sreq.Fields = []string{"_id"}
	sres, _ := mi.index.Search(sreq)
	batch := mi.index.NewBatch()
	for _, hit := range sres.Hits {
		batch.Delete(hit.Id)
	}
	batch.Execute()
	_, _ = mi.db.Exec("DELETE FROM vectors WHERE path = ?", path)
	chunks, err := chunk.ChunksFromFile(path)
	if err != nil {
		return err
	}
	batchSize := 32
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batchChunks := chunks[i:end]
		texts := make([]string, len(batchChunks))
		for k, c := range batchChunks {
			texts[k] = c.Content
		}
		embs, err := mi.provider.Embed(texts)
		if err != nil {
			log.Printf("embed failed %s: %v", path, err)
			continue
		}
		indexBatch := mi.index.NewBatch()
		for k := range batchChunks {
			c := batchChunks[k]
			id := fmt.Sprintf("%s:%d:%d", c.Path, c.StartLine, c.EndLine)
			doc := struct {
				Path      string `json:"path"`
				StartLine int    `json:"start_line"`
				EndLine   int    `json:"end_line"`
				Content   string `json:"content"`
			}{c.Path, c.StartLine, c.EndLine, c.Content}
			indexBatch.Index(id, doc)
			store.InsertVector(mi.db, id, c.Path, c.StartLine, c.EndLine, c.Content, embs[k])
		}
		indexBatch.Execute()
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}