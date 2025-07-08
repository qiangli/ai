package bridge

import (
	"database/sql"
	"fmt"
)

func createPostgresClient(credentials *Credentials) (*sql.DB, error) {
	host := credentials.Host
	port := credentials.Port
	user := credentials.User
	pwd := credentials.Password
	db := credentials.Database
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if user == "" {
		user = "postgres"
	}
	if db == "" {
		db = "postres"
	}
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pwd, db)
	return sql.Open("postgres", connStr)
}

func createMySQLClient(credentials *Credentials) (*sql.DB, error) {
	host := credentials.Host
	port := credentials.Port
	user := credentials.User
	pwd := credentials.Password
	db := credentials.Database
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "3306"
	}
	if user == "" {
		user = "root"
	}
	if db == "" {
		db = "mydb"
	}
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pwd, host, port, db)
	return sql.Open("mysql", connStr)
}
