package api

// import (
// 	"fmt"
// )

// type DBCred struct {
// 	Host     string `mapstructure:"host"`
// 	Port     string `mapstructure:"port"`
// 	Username string `mapstructure:"username"`
// 	Password string `mapstructure:"password"`
// 	DBName   string `mapstructure:"name"`
// }

// // DSN returns the data source name for connecting to the database.
// func (d *DBCred) DSN() string {
// 	host := d.Host
// 	if host == "" {
// 		host = "localhost"
// 	}
// 	port := d.Port
// 	if port == "" {
// 		port = "5432"
// 	}
// 	dbname := d.DBName
// 	if dbname == "" {
// 		dbname = "postgres"
// 	}
// 	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, d.Username, d.Password, dbname)
// }

// func (d *DBCred) IsValid() bool {
// 	return d.Username != "" && d.Password != ""
// }
