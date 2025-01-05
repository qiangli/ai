package tool

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"

	"github.com/qiangli/ai/internal/db"
)

var DBTools = []openai.ChatCompletionToolParam{
	define("db_query",
		"Run query against the database",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]string{
					"type":        "string",
					"description": "SELECT SQL query to run",
				},
			},
			"required": []string{"query"},
		}),
	define("db_version",
		"Gather database version information",
		nil,
	),
	define("db_all_databases",
		"List all available databases",
		nil,
	),
}

func runDbTool(cfg *Config, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	getStr := func(key string) (string, error) {
		val, ok := props[key]
		if !ok {
			return "", fmt.Errorf("missing property: %s", key)
		}
		str, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("property '%s' must be a string", key)
		}
		return str, nil
	}

	switch name {
	case "db_query":
		query, err := getStr("query")
		if err != nil {
			return "", err
		}
		return db.RunQuery(cfg.DBConfig, ctx, query)
	case "db_version":
		return db.RunQuery(cfg.DBConfig, ctx, "SELECT version()")

	case "db_all_databases":
		return db.RunQuery(cfg.DBConfig, ctx, "SELECT datname FROM pg_database WHERE datistemplate = false AND datallowconn = true")
	}

	return "", nil
}
