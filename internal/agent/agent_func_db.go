package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/swarm"
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

func init() {
	funcRegistry["db_query"] = sqlDBQuery
	funcRegistry["db_version"] = sqlDBVersion
	funcRegistry["db_all_databases"] = sqlDBAllDatabases
	funcRegistry["db_all_tables"] = sqlDBAllTables
	funcRegistry["db_all_views"] = sqlDBAllViews
	funcRegistry["db_all_sequences"] = sqlDBAllSequences
	funcRegistry["db_all_columns"] = sqlDBAllColumns
}

func sqlQuery(ctx context.Context, cred *api.DBCred, query string) (*api.Result, error) {
	result, err := db.RunQuery(cred, ctx, query)
	if err != nil {
		result = fmt.Sprintf("Error: %s", err.Error())
	}
	return &api.Result{
		Value: result,
	}, nil
}

func sqlDBQuery(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	query, err := swarm.GetStrProp("query", args)
	if err != nil {
		return nil, err
	}
	return sqlQuery(ctx, agent.Vars.DBCred, query)
}

func sqlDBVersion(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	return sqlQuery(ctx, agent.Vars.DBCred, versionQuery)
}

func sqlDBAllDatabases(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	return sqlQuery(ctx, agent.Vars.DBCred, allDatabasesQuery)
}

func sqlDBAllTables(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	return sqlQuery(ctx, agent.Vars.DBCred, allTablesQuery)
}

func sqlDBAllViews(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	return sqlQuery(ctx, agent.Vars.DBCred, allViewsQuery)
}
func sqlDBAllSequences(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	return sqlQuery(ctx, agent.Vars.DBCred, allSequencesQuery)
}

func sqlDBAllColumns(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	schema, err := swarm.GetStrProp("schema", args)
	if err != nil {
		return nil, err
	}
	table, err := swarm.GetStrProp("table", args)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(allColumnsQuery, schema, table)
	return sqlQuery(ctx, agent.Vars.DBCred, query)
}
