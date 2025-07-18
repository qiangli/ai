package db

import (
	"os"
	"testing"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/xwb1989/sqlparser"
)

func TestSqlParser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	files := []string{
		"testdata/0000_keen_devos.sql",
	}

	for _, file := range files {
		query, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		stmt, err := sqlparser.Parse(string(query))
		if err != nil {
			t.Logf("failed to parse %s: %v\n", file, err)
			// t.Fail()
		} else {
			t.Logf("parsed %s: %s\n", file, stmt)
		}
	}
}

func TestParser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	files := []string{
		"testdata/0000_keen_devos.sql",
	}

	for _, file := range files {
		query, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		stmts, err := parser.Parse(string(query))

		if err != nil {
			t.Logf("failed to parse %s: %v\n", file, err)
			// t.Fail()
		} else {
			t.Logf("parsed %s: %v\n", file, len(stmts))
		}
	}
}
