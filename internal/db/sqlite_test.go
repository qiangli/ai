package db

import (
	"path/filepath"
	"testing"
)

func TestSqliteNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ddl := `CREATE TABLE website (
			"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
			"title" TEXT,
			"url" TEXT,
			"content" TEXT		
		  );`

	ds, err := New(dbPath)
	if err != nil {
		t.Errorf("expected not nil")
	}
	defer ds.Close()

	result, err := ds.CreateTable(ddl)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("result: %v", result)
	sql := `INSERT INTO website(title, url, content) VALUES (?, ?, ?)`

	ds.Execute(sql, "Google", "https://google.com", "Google search engine")
	ds.Execute(sql, "Microsoft Bing", "https://bing.com", "Bing search engine")
	ds.Execute(sql, "DuckDuckGo", "https://duckduckgo.com", "DuckDuckGo search engine")

	row, err := ds.Query("SELECT * FROM website ORDER BY title")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	defer row.Close()

	for row.Next() {
		var id int
		var title string
		var url string
		var content string
		row.Scan(&id, &title, &url, &content)
		t.Logf("id: %v, title: %v, url: %v, content: %v", id, title, url, content)
	}
}
