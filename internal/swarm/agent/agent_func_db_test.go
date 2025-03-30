package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/qiangli/ai/api"
)

func TestSqlQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"Test sqlQuery", versionQuery, "PostgreSQL 15.8"},
		{"Test allDatabasesQuery", allDatabasesQuery, "DATNAME NAME\npostgres\n"},
		{"Test allTablesQuery", allTablesQuery, "SCHEMANAME NAME\tTABLENAME NAME"},
		{"Test allViewsQuery", allViewsQuery, "SCHEMANAME NAME	VIEWNAME NAME"},
		{"Test allSequencesQuery", allSequencesQuery, "SCHEMANAME NAME	SEQUENCENAME NAME"},
		{"Test allColumnsQuery", allColumnsQuery, "TABLE_SCHEMA NAME	TABLE_NAME NAME	COLUMN_NAME NAME"},
		{"Test allColumnsQuery with schema", allColumnsQuery, "TABLE_SCHEMA NAME	TABLE_NAME NAME	COLUMN_NAME NAME"},
	}

	ctx := context.Background()

	cred := &api.DBCred{
		Host:     "localhost",
		Port:     "5432",
		Username: "",
		Password: "",
		DBName:   "postgres",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := sqlQuery(ctx, cred, test.query)
			if err != nil {
				t.Errorf("Error: %s", err.Error())
			}
			t.Logf("Result:\n%s\n", result.Value)
			if !strings.Contains(result.Value, test.expected) {
				t.Errorf("Expected %s, got %s", test.expected, result.Value)
			}
		})
	}
}
