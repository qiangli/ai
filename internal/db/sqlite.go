// https://pkg.go.dev/modernc.org/sqlite
// https://sqlite.org/docs.html
package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type DataStore struct {
	db *sql.DB
}

func NewDB(dsn string) (*DataStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	return &DataStore{db: db}, nil
}

func (r *DataStore) CreateTable(ddl string) (sql.Result, error) {
	statement, err := r.db.Prepare(ddl)
	if err != nil {
		return nil, err
	}
	return statement.Exec()
}

func (r *DataStore) Close() error {
	return r.db.Close()
}

func (r *DataStore) Execute(sql string, args ...any) (sql.Result, error) {
	statement, err := r.db.Prepare(sql)
	if err != nil {
		return nil, err
	}
	return statement.Exec(args...)
}

func (r *DataStore) Query(sql string, args ...any) (*sql.Rows, error) {
	statement, err := r.db.Prepare(sql)
	if err != nil {
		return nil, err
	}
	return statement.Query(args...)
}
