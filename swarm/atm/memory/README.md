# ATM Memory - Semantic Search Tools

## Overview

Hybrid semantic search (Bleve BM25 + vector cosine) over Markdown workspace files.

Supports local/random, OpenAI, Gemini embeddings.

SQLite vector store (cgo-free), fsnotify watch.

## Architecture

```
Workspace (memory/*.md, MEMORY.md)
  ↓ chunk (~400 tok, 80 overlap, line-preserve)
  ↓ embed batch (768 dim)
  ↙                 ↘
Bleve BM25       SQLite vectors (full scan sim)
  ↓                  ↓
Text top K     Vector top K
  ↓ merge weighted (0.7v + 0.3t normalized)
Top N results
```

## Configuration (MemoryConfig)

| Field | Default | Desc |
|-------|---------|------|
| enabled | false | Enable memory |
| provider | "local" | "local","openai","gemini" |
| model | "" | e.g. "text-embedding-3-small" |
| fallback | "" | fallback provider |
| store_path | temp | bleve/ + vectors.db |
| extra_paths | [] | globs |
| query.max_results | 5 |  |
| query.hybrid.enabled | true |  |
| query.hybrid.vector_weight | 0.7 |  |
| query.hybrid.text_weight | 0.3 |  |
| query.hybrid.candidate_multiplier | 3 |  |
| sync.watch | [] | dirs to watch recursive |
| sync.debounce | "1.5s" |  |
| cache.enabled | false | TTL | 

## Usage

### Go API

```go
cfg := memory.MemoryConfig{Enabled: true, Provider: "local", StorePath: "/tmp/mem"}
res, _ := memory.Search("AI agent", cfg)
content, _ := memory.Get("memory/2024.md", 1, 20, cfg)
memory.IndexWorkspace("/workspace", cfg) // initial index
```

### CLI

```bash
cd atm/memory
go mod tidy
go build -o memorytool ./cmd/memorytool
echo '{"query":"test","config":{"enabled":true,"provider":"local","store_path":"/tmp/testmem"}}' | ./memorytool search
./memorytool get --json '{"path":"test.md","from_line":1,"lines":10}'
```

Index first: go run api.go or manual IndexWorkspace.

## YAML Tools for Claw Agents

Build `./memorytool`, update command path in tools/*.yaml.

Config uses /tmp/atm-memory-store (agent ensure writable).

Pre-index workspace before agent run.

## Tests & Build

```bash
go test ./...   # all pass, local embed
go vet ./...
go build ./cmd/memorytool
```

## Design

- Chunk: char-based ~1600 chars/target, line ranges for Get()
- Embed: retry 3x exp backoff
- Hybrid: union candidates, norm by max in pool, weighted sum
- Incremental index per file on watch
- Dim: 768 (text-embedding-3-small)

Refs:
- OpenClaw /poc/openclaw/docs/concepts/memory.md
- Bleve: github.com/blevesearch/bleve
- SQLite: modernc.org/sqlite