package db

import (
	"database/sql"
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

	ds, err := NewDB(dbPath)
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

func TestChatHistory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ddl := `CREATE TABLE chats (
			"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			"uid" TEXT,
			"created" timestamp,
			"updated" timestamp,
			"accessed" timestamp,
			"content" TEXT		
		  );`

	ds, err := NewDB(dbPath)
	if err != nil {
		t.Errorf("expected not nil")
	}
	defer ds.Close()

	result, err := ds.CreateTable(ddl)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("result: %v", result)

	query := `
        INSERT INTO chats (uid, created, updated, accessed, content)
        VALUES (?, ?, ?, ?, ?)
    `

	insert := func(content string) (sql.Result, error) {
		uid := genUID()
		created := currentTS()

		return ds.Execute(query, uid, created, created, created, content)
	}

	insert("Hello, World!")
	insert("How are you?")
	insert("I'm fine, thank you!")
	insert("Goodbye!")

	row, err := ds.Query("SELECT * FROM chats ORDER BY id desc")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	defer row.Close()

	for row.Next() {
		var id int
		var uid string
		var created, updated, accessed string
		var content string

		if err := row.Scan(&id, &uid, &created, &updated, &accessed, &content); err != nil {
			t.Errorf("failed to scan row: %v", err)
			return
		}
		t.Logf("id: %v, uid: %v, created: %v, updated: %v, accessed: %v, content: %v", id, uid, created, updated, accessed, content)
	}
}
