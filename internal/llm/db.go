package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"

	"github.com/qiangli/ai/internal/db"
)

const versionQuery = `"SELECT version()"`

const allDatabasesQuery = `SELECT datname FROM pg_database WHERE datistemplate = false AND datallowconn = true`

const allTablesQuery = `
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema');
`

const allViewsQuery = `
SELECT schemaname, viewname
FROM pg_views
WHERE schemaname NOT IN ('pg_catalog', 'information_schema');
`

const allSequencesQuery = `
SELECT schemaname, sequencename
FROM pg_sequences
WHERE schemaname NOT IN ('pg_catalog', 'information_schema');
`

const allColumnsQuery = `
SELECT table_schema, table_name, column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_schema = '%s' AND table_name = '%s'
AND table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY table_schema, table_name, ordinal_position;
`

var dbTools = []openai.ChatCompletionToolParam{
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
	define("db_all_tables",
		"List all tables in the database",
		nil,
	),
	define("db_all_views",
		"List all views in the database",
		nil,
	),
	define("db_all_sequences",
		"List all sequences in the database",
		nil,
	),
	define("db_all_columns",
		"List all columns in a table",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"schema": map[string]string{
					"type":        "string",
					"description": "Schema name",
				},
				"table": map[string]string{
					"type":        "string",
					"description": "Table name",
				},
			},
			"required": []string{"schema", "table"},
		}),
}

func runDbTool(cfg *ToolConfig, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	getStr := func(key string) (string, error) {
		return getStrProp(key, props)
	}

	switch name {
	case "db_query":
		query, err := getStr("query")
		if err != nil {
			return "", err
		}
		return db.RunQuery(cfg.DBConfig, ctx, query)
	case "db_version":
		return db.RunQuery(cfg.DBConfig, ctx, versionQuery)
	case "db_all_databases":
		return db.RunQuery(cfg.DBConfig, ctx, allDatabasesQuery)
	case "db_all_tables":
		return db.RunQuery(cfg.DBConfig, ctx, allTablesQuery)
	case "db_all_views":
		return db.RunQuery(cfg.DBConfig, ctx, allViewsQuery)
	case "db_all_sequences":
		return db.RunQuery(cfg.DBConfig, ctx, allSequencesQuery)
	case "db_all_columns":
		schema, err := getStr("schema")
		if err != nil {
			return "", err
		}
		table, err := getStr("table")
		if err != nil {
			return "", err
		}
		query := fmt.Sprintf(allColumnsQuery, schema, table)
		return db.RunQuery(cfg.DBConfig, ctx, query)
	}

	return "", nil
}

func GetDBTools() []openai.ChatCompletionToolParam {
	return dbTools
}
