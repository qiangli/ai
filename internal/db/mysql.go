package db

import (
	"context"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/go-mysql-server/sql"

	"github.com/qiangli/ai/internal/log"
)

// https://github.com/dolthub/go-mysql-server
// > mysql --host=localhost --port=3306 --user=root mydb --execute="SELECT * FROM mytable;"
// var (
// 	// dbname    = "mydb"
// 	// tableName = "mytable"

// 	// address   = "localhost"
// 	// port      = 3306
// )

func StartMySQL(address string, dbname string) {
	pro := createTestDatabase(dbname)
	engine := sqle.NewDefault(pro)

	session := memory.NewSession(sql.NewBaseSession(), pro)
	ctx := sql.NewContext(context.Background(), sql.WithSession(session))
	ctx.SetCurrentDatabase(dbname)

	config := server.Config{
		Protocol: "tcp",
		Address:  address,
	}
	s, err := server.NewServer(config, engine, sql.NewContext, memory.NewSessionBuilder(pro), nil)
	if err != nil {
		log.Errorf("failed to create %v\n", err)
		return
	}

	log.Infof("MySQL listening on %s...\n", address)

	if err = s.Start(); err != nil {
		log.Errorf("failed to start %v\n", err)
		return
	}
}

func createTestDatabase(dbname string) *memory.DbProvider {
	db := memory.NewDatabase(dbname)
	db.BaseDatabase.EnablePrimaryKeyIndexes()

	pro := memory.NewDBProvider(db)

	return pro
}
