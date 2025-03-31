package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/swarm/api"
)

func sqlQuery(ctx context.Context, cred *api.DBCred, query string) (string, error) {
	result, err := db.RunQuery(cred, ctx, query)
	if err != nil {
		return "", err
	}
	return result, nil
}

func dbCred(vars *api.Vars, args map[string]any) (*api.DBCred, error) {
	cred := vars.Config.DBCred.Clone()
	if !cred.IsValid() {
		return nil, fmt.Errorf("invalid database credentials")
	}

	// default if not set
	db, _ := GetStrProp("database", args)
	if db != "" {
		cred.DBName = db
	}
	return cred, nil
}
