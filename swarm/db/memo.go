package db

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/qiangli/ai/swarm/api"
)

type Message = api.Message
type MemOption = api.MemOption

type MemoryStore struct {
	ds *DataStore
}

const layout = "2006-01-02 15:04:05.999999 -0700 MST"

func OpenMemoryStore(base string, file string) (*MemoryStore, error) {
	const ddl = `CREATE TABLE IF NOT EXISTS chats (
			"id" TEXT NOT NULL,		
			"session" TEXT,
			"created" DATETIME,
			"content_type" TEXT,
			"content" TEXT,
			"role" TEXT,
			"sender" TEXT		
		  );`

	dbPath := filepath.Join(base, file)

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
        INSERT INTO chats (id, session, created, content_type, content, role, sender)
        VALUES (?, ?, ?, ?, ?, ?, ?)`

	for _, v := range messages {
		if _, err := m.ds.Execute(query, v.ID, v.Session, v.Created, v.ContentType, v.Content, v.Role, v.Sender); err != nil {
			return err
		}
	}

	return nil
}

func (m *MemoryStore) Load(opt *MemOption) ([]*Message, error) {
	var defaultRoles = []string{"assistant", "user"}
	// if opt == nil {
	// 	opt = &api.MemOption{
	// 		MaxHistory: 3,
	// 		MaxSpan:    1440,
	// 		Offset:     0,
	// 	}
	// }
	if opt == nil || opt.MaxHistory == 0 || opt.MaxSpan == 0 {
		return []*Message{}, nil
	}
	if len(opt.Roles) == 0 {
		opt.Roles = defaultRoles
	}

	var messages []*Message
	maxSpan := time.Now().Add(-time.Duration(opt.MaxSpan) * time.Minute).Unix()

	rolePlaceholders := strings.Repeat("?,", len(opt.Roles))
	rolePlaceholders = rolePlaceholders[:len(rolePlaceholders)-1]

	// var query = fmt.Sprintf(`
	// 	SELECT id, session, created, content_type, content, role, sender
	// 	FROM chats
	// 	WHERE created >= ?
	// 	AND role IN (%s)
	// 	ORDER BY created ASC
	// 	LIMIT ? OFFSET ?
	// 	`, rolePlaceholders)

	var query = fmt.Sprintf(`
		SELECT id, session, MAX(created) as created, content_type, content, role, sender
		FROM (
			SELECT * 
			FROM chats
			WHERE created >= ?
			AND role IN (%s)
			ORDER BY created DESC
		)
		GROUP BY id
		ORDER BY created ASC
		LIMIT ? OFFSET ?
		`, rolePlaceholders)

	// Build the args slice
	var args []any
	args = append(args, maxSpan)
	for _, role := range opt.Roles {
		args = append(args, role)
	}
	args = append(args, opt.MaxHistory, opt.Offset)

	rows, err := m.ds.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		var created string
		if err := rows.Scan(&msg.ID, &msg.Session, &created, &msg.ContentType, &msg.Content, &msg.Role, &msg.Sender); err != nil {
			return nil, err
		}
		if idx := strings.LastIndex(created, " m="); idx != -1 {
			created = created[:idx]
		}
		msg.Created, err = time.Parse(layout, created)
		if err != nil {
			return nil, err
		}

		messages = append(messages, &msg)
	}
	return messages, nil
}

func (m *MemoryStore) Get(id string) (*Message, error) {
	const query = `
		SELECT id, session, created, content_type, content, role, sender
		FROM chats
		WHERE id = ?`

	row, err := m.ds.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var msg Message
	if row.Next() {
		var created string
		if err := row.Scan(&msg.ID, &msg.Session, &created, &msg.ContentType, &msg.Content, &msg.Role, &msg.Sender); err != nil {
			return nil, err
		}
		if idx := strings.LastIndex(created, " m="); idx != -1 {
			created = created[:idx]
		}
		msg.Created, err = time.Parse(layout, created)
		if err != nil {
			return nil, err
		}

	}
	return &msg, nil
}
