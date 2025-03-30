package db

import (
	"context"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

// https://github.com/jfcg/sorty/issues/6

func TestQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	ctx := context.Background()
	cfg := &api.DBCred{
		Host:     "localhost",
		Port:     "5432",
		Username: "",
		Password: "",
		DBName:   "",
	}

	tests := []struct {
		query string
	}{
		{"SELECT version()"},
		{"SELECT datname FROM pg_database WHERE datistemplate = false AND datallowconn = true"},
		{"SELECT * FROM analyses ORDER BY id ASC limit 0"},
	}

	for _, tt := range tests {
		out, err := RunQuery(cfg, ctx, tt.query)
		if err != nil {
			t.Errorf("Query error: %v", err)
			return
		}
		t.Logf("Query out:\n%v", out)
	}
}
