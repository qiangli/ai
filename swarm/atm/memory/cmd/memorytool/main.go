package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"atm/memory"
)

type SearchReq struct {
	Query  string           `json:"query"`
	Config memory.MemoryConfig `json:"config"`
}

type GetReq struct {
	Path     string `json:"path"`
	FromLine int    `json:"from_line"`
	Lines    int    `json:"lines"`
	Config   memory.MemoryConfig `json:"config"`
}

type SearchResp struct {
	Results []memory.SearchResult `json:"results"`
}

type GetResp struct {
	Content string `json:"content"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: memorytool search|get [--json JSON]")
	}
	cmd := os.Args[1]
	flag.Parse()
	var data []byte
	if flag.NArg() > 2 && flag.Arg(1) == "--json" {
		data = []byte(flag.Arg(2))
	} else {
		data, _ = io.ReadAll(os.Stdin)
	}
	switch cmd {
	case "search":
		var req SearchReq
		json.Unmarshal(data, &req)
		res, err := memory.Search(req.Query, req.Config)
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(os.Stdout).Encode(SearchResp{Results: res})
	case "get":
		var req GetReq
		json.Unmarshal(data, &req)
		cont, err := memory.Get(req.Path, req.FromLine, req.Lines, req.Config)
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(os.Stdout).Encode(GetResp{Content: cont})
	}
}