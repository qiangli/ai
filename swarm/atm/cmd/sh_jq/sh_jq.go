package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/itchyny/gojq"
	"io"
	"log"
	"os"
)

var (
	expr    = flag.String("e", "", "expression")
	confirm = flag.Bool("confirm", false, "Confirm destructive actions (default false)")
	dryRun  = flag.Bool("dry-run", true, "Run in dry-run mode by default; no destructive actions executed")
	raw     = flag.Bool("raw-output", false, "Output raw strings")
	compact = flag.Bool("compact-output", false, "Compact output")
)

func main() {
	flag.Parse()
	if *expr == "" {
		fmt.Fprintln(os.Stderr, "no expression provided; use -e or --expr")
		os.Exit(2)
	}

	// Safety: if dry-run and confirm both set in conflicting ways, prefer requiring explicit confirm for destructive ops.
	// For now we only enforce: changes require --confirm. This wrapper cannot detect all destructive intents in jq program,
	// so it will ask if expression contains the word "delete" or "remove" or "del(" as a heuristic.
	heur := *expr
	if (*dryRun || !*confirm) && (containsDestructiveHeuristic(heur)) {
		fmt.Fprintln(os.Stderr, "Destructive expression detected. Pass --confirm to execute or run without --dry-run. Exiting.")
		os.Exit(3)
	}

	// Read input
	var r io.Reader = os.Stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatalf("open input: %v", err)
		}
		defer f.Close()
		r = f
	}
	data, err := io.ReadAll(r)
	if err != nil {
		log.Fatalf("read input: %v", err)
	}

	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		// try as raw string
		input = string(data)
	}
	query, err := gojq.Parse(*expr)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		log.Fatalf("compile: %v", err)
	}
	iter := code.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalf("execution error: %v", err)
		}
		if *raw {
			fmt.Println(v)
		} else {
			b, _ := gojq.Marshal(v)
			if *compact {
				fmt.Println(string(b))
			} else {
				fmt.Printf("%s\n", b)
			}
		}
	}
}

func containsDestructiveHeuristic(s string) bool {
	if s == "" {
		return false
	}
	if contains(s, "delete") || contains(s, "remove") || contains(s, "del(") {
		return true
	}
	return false
}

func contains(hay, needle string) bool {
	return len(hay) >= len(needle) && (indexOf(hay, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
