package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
}

// DSN returns the data source name for connecting to the database.
func (d *DBConfig) DSN() string {
	host := d.Host
	if host == "" {
		host = "localhost"
	}
	port := d.Port
	if port == "" {
		port = "5432"
	}
	dbname := d.DBName
	if dbname == "" {
		dbname = "postgres"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, d.Username, d.Password, dbname)
}

func (d *DBConfig) IsValid() bool {
	return d.Username != "" && d.Password != ""
}

type Queryable interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

type Scanable interface {
	Scan(dest ...interface{}) error
}

type Scanner[T any] func(Scanable) (T, error)

func ScanSlice[T any](scanner Scanner[T], rows *sql.Rows) ([]T, error) {
	results := make([]T, 0)
	for rows.Next() {
		x, err := scanner(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, x)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

type PGVersion struct {
	Version string
}

func scanPGVersion(s Scanable) (*PGVersion, error) {
	var e PGVersion
	if err := s.Scan(&e.Version); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &e, nil
}

func RetrievePGVersion(ctx context.Context, q Queryable) (*PGVersion, error) {
	return scanPGVersion(q.QueryRowContext(ctx, `
	SELECT version()
	`))
}

type PGDatabase struct {
	Datname string
}

func scanPGDatabase(s Scanable) (*PGDatabase, error) {
	var e PGDatabase
	if err := s.Scan(&e.Datname); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &e, nil
}

func RetrieveDatabases(ctx context.Context, q Queryable) ([]*PGDatabase, error) {
	const query = `
	SELECT datname FROM pg_database WHERE datistemplate = false AND datallowconn = true
	`
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*PGDatabase, 0)
	for rows.Next() {
		e, err := scanPGDatabase(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return results, nil
}

func Connect(db *DBConfig) (*sql.DB, error) {
	return sql.Open("postgres", db.DSN())
}

func Ping(db *DBConfig) error {
	pg, err := Connect(db)
	if err != nil {
		return err
	}
	defer pg.Close()
	return pg.Ping()
}

func RunSelectQuery(ctx context.Context, q Queryable, query string) (*sql.Rows, error) {
	return q.QueryContext(ctx, query)
}

func RunQuery(cfg *DBConfig, ctx context.Context, query string) (string, error) {
	pg, err := Connect(cfg)
	if err != nil {
		return "", err
	}
	defer pg.Close()

	rows, err := RunSelectQuery(ctx, pg, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return "", err
	}

	columnHeaders := make([]string, len(columnTypes))
	for i, colType := range columnTypes {
		// Column name and type in format "<name> <type>"
		columnHeaders[i] = fmt.Sprintf("%s %s", strings.ToUpper(colType.Name()), strings.ToUpper(colType.DatabaseTypeName()))
	}

	values := make([]interface{}, len(columnTypes))
	valuePtrs := make([]interface{}, len(columnTypes))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	results := make([]string, 0)
	results = append(results, strings.Join(columnHeaders, "\t"))

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}

		var rowStrings []string
		for _, val := range values {
			if val != nil {
				rowStrings = append(rowStrings, fmt.Sprintf("%s", val))
			} else {
				rowStrings = append(rowStrings, "NULL")
			}
		}

		results = append(results, strings.Join(rowStrings, "\t"))
	}

	err = rows.Err()
	if err != nil {
		return "", err
	}

	return strings.Join(results, "\n"), nil
}
