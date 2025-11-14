package db

import (
	"path/filepath"
	"time"

	"github.com/qiangli/ai/swarm/api"
)

type Message = api.Message
type MemOption = api.MemOption

type MemoryStore struct {
	ds *DataStore
}

func OpenMemoryStore(cfg *api.AppConfig) (*MemoryStore, error) {
	const ddl = `CREATE TABLE IF NOT EXISTS chats (
			"id" TEXT NOT NULL,		
			"chat_id" TEXT,
			"created" DATETIME,
			"content_type" TEXT,
			"content" TEXT,
			"role" TEXT,
			"sender" TEXT		
		  );`

	dbPath := filepath.Join(cfg.Base, "memory.db")

	ds, err := NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := ds.CreateTable(ddl); err != nil {
		return nil, err
	}

	return &MemoryStore{ds: ds}, nil
}

func (m *MemoryStore) Close() error {
	return m.ds.Close()
}

func (m *MemoryStore) Save(messages []*Message) error {
	const query = `
        INSERT INTO chats (id, chat_id, created, content_type, content, role, sender)
        VALUES (?, ?, ?, ?, ?, ?, ?)`

	for _, v := range messages {
		if _, err := m.ds.Execute(query, v.ID, v.ChatID, v.Created, v.ContentType, v.Content, v.Role, v.Sender); err != nil {
			return err
		}
	}

	return nil
}

func (m *MemoryStore) Load(opt *MemOption) ([]*Message, error) {
	const query = `
		SELECT id, chat_id, created, content_type, content, role, sender
		FROM chats
		WHERE created >= ? ORDER BY created ASC LIMIT ?`

	var messages []*Message
	maxSpan := time.Now().Add(-time.Duration(opt.MaxSpan) * time.Minute).Unix()
	rows, err := m.ds.Query(query, maxSpan, opt.MaxHistory)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.Created, &msg.ContentType, &msg.Content, &msg.Role, &msg.Sender); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}

func (m *MemoryStore) Get(id string) (*Message, error) {
	const query = `
		SELECT id, chat_id, created, content_type, content, role, sender
		FROM chats
		WHERE id = ?`
	var msg Message
	row, err := m.ds.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if row.Next() {
		if err := row.Scan(&msg.ID, &msg.ChatID, &msg.Created, &msg.ContentType, &msg.Content, &msg.Role, &msg.Sender); err != nil {
			return nil, err
		}
	}
	return &msg, nil
}
