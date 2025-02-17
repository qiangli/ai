package db

import (
	"context"
	"fmt"
	"regexp"

	"github.com/qiangli/ai/internal/api"
)

func GetDBInfo(cfg *api.DBCred) (map[string]string, error) {
	pg, err := Connect(cfg)
	if err != nil {
		return nil, err
	}
	defer pg.Close()

	ctx := context.Background()
	dbs, err := RetrieveDatabases(ctx, pg)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, db := range dbs {
		names = append(names, db.Datname)
	}

	pgVersion, err := RetrievePGVersion(ctx, pg)
	if pgVersion == nil {
		return nil, err
	}

	dialect, version := splitVersion(pgVersion.Version)

	info := map[string]string{
		"PG":        pgVersion.Version,
		"Dialect":   dialect,
		"Version":   version,
		"Databases": fmt.Sprintf("Available databases: %v", names),
	}
	return info, nil
}

func splitVersion(s string) (string, string) {
	// postgres version string format: PostgreSQL 15.8 on ...
	re := regexp.MustCompile(`(\w+) (\d+\.\d+(\.\d+)?)`)
	match := re.FindStringSubmatch(s)

	if len(match) > 2 {
		dialect := match[1]
		version := match[2]
		return dialect, version
	}
	return "", ""
}
