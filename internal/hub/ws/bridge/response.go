package bridge

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type Credentials struct {
	ID         int    `json:"id"`
	User       string `json:"user"`
	Password   string `json:"password"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	Database   string `json:"database"`
	Autofilled bool   `json:"autofilled"`
}

func response(msg *Message, ctx *sync.Map) (map[string]any, error) {
	switch msg.Action {
	case "open":
		return openConnection(msg, ctx)
	case "exec":
		return executeSQL(msg, ctx)
	case "close":
		return closeConnection(msg, ctx)
	case "get_credentials":
		return getCredentials(msg, ctx)
	default:
		return nil, errors.New("unknown action: " + msg.Action)
	}
}

func openConnection(msg *Message, ctx *sync.Map) (map[string]any, error) {
	cred := msg.Credentials

	if cred == nil {
		// return nil, fmt.Errorf("missing credentials")
		cred = &Credentials{}
	}

	var db *sql.DB
	var err error

	switch msg.DB {
	case "postgres":
		db, err = createPostgresClient(cred)
	case "mysql":
		db, err = createMySQLClient(cred)
	default:
		return nil, errors.New("database " + msg.DB + " not recognized")
	}

	if err != nil {
		return nil, err
	}

	ctx.Store("db", db)
	return map[string]any{"ready": true}, nil
}

func executeSQL(msg *Message, ctx *sync.Map) (map[string]any, error) {
	dbInterface, ok := ctx.Load("db")
	if !ok {
		return nil, errors.New("database connection is not available")
	}
	db, ok := dbInterface.(*sql.DB)
	if !ok {
		return nil, errors.New("invalid database connection")
	}

	queries := []string{msg.SQL}
	var results = map[string]any{}

	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		if strings.HasPrefix(strings.ToUpper(query), "SELECT") {
			v, err := executeQuery(db, query)
			if err != nil {
				return nil, err
			}
			results = v
		} else {
			result, err := db.Exec(query)
			if err != nil {
				return nil, fmt.Errorf("error executing query: %v", err)
			}

			_, err = result.RowsAffected()
			if err != nil {
				return nil, fmt.Errorf("error getting affected rows: %v", err)
			}

			results = map[string]any{"rows": []string{}, "fields": []string{}}
		}
	}

	return map[string]any{"results": results}, nil
}

func executeQuery(db *sql.DB, query string) (map[string]any, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	fields := make([]any, len(cols))
	for i, col := range cols {
		fields[i] = map[string]any{
			"Name": col,
		}
	}

	var rowResults = [][]string{}
	rowCount := 0
	for rows.Next() {
		values, err := rowsValues(rows, len(cols))
		if err != nil {
			return nil, err
		}
		rowResults = append(rowResults, values)
		rowCount++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"command":  "SELECT",
		"rowCount": rowCount,
		"rows":     rowResults,
		"fields":   fields,
	}, nil
}

func rowsValues(rows *sql.Rows, numCols int) ([]string, error) {
	dest := make([]interface{}, numCols)
	for i := range dest {
		dest[i] = new(interface{})
	}

	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}

	result := make([]string, numCols)
	for i, val := range dest {
		if b, ok := (*(val.(*interface{}))).([]byte); ok {
			result[i] = string(b)
		} else {
			result[i] = fmt.Sprintf("%v", *(val.(*interface{})))
		}
	}
	return result, nil
}

func closeConnection(msg *Message, ctx *sync.Map) (map[string]any, error) {
	dbInterface, ok := ctx.Load("db")
	if !ok {
		return nil, errors.New("no database connection found in context")
	}

	db, ok := dbInterface.(*sql.DB)
	if !ok {
		return nil, errors.New("invalid database connection")
	}

	// Close the database connection
	err := db.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing database connection: %v", err)
	}

	// Remove the connection from context
	ctx.Delete("db")

	return map[string]any{"closed": true}, nil
}

func getCredentials(msg *Message, ctx *sync.Map) (map[string]any, error) {
	switch msg.DB {
	case "postgres":
		password := os.Getenv("PGPASSWORD")
		if password == "" {
			password = "password"
		}
		return map[string]any{
			"host":     "localhost",
			"port":     5432,
			"database": "postgres",
			"user":     "postgres",
			"password": password,
		}, nil
	case "mysql":
		return map[string]any{
			"host":     "localhost",
			"port":     3306,
			"database": "mydb",
			"user":     "root",
			"password": "",
		}, nil
	}
	return nil, fmt.Errorf("unpoorted %s", msg.DB)
}
