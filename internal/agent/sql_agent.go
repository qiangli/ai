package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type SqlAgent struct {
	config *llm.Config

	Role   string
	Prompt string
}

func NewSqlAgent(cfg *llm.Config, role, prompt string) (*SqlAgent, error) {
	if role == "" {
		role = "system"
	}

	if prompt == "" {
		var err error
		info, err := getDBInfo(cfg.Sql.DBConfig)
		if err != nil {
			return nil, err
		}
		prompt, err = resource.GetSqlSystemRoleContent(info)
		if err != nil {
			return nil, err
		}
	}

	// Set up the tools
	cfg.Tools = llm.GetDBTools()

	agent := SqlAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *SqlAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	content, err := llm.Send(r.config, ctx, r.Role, r.Prompt, in.Input())
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "SQL",
		Content: content,
	}, nil
}

func getDBInfo(cfg *db.DBConfig) (*resource.DBInfo, error) {
	pg, err := db.Connect(cfg)
	if err != nil {
		return nil, err
	}
	defer pg.Close()

	ctx := context.Background()
	dbs, err := db.RetrieveDatabases(ctx, pg)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, db := range dbs {
		names = append(names, db.Datname)
	}

	version, err := db.RetrievePGVersion(ctx, pg)
	if version == nil {
		return nil, err
	}

	info := resource.DBInfo{
		Version:     version.Version,
		ContextData: fmt.Sprintf("Available databases: %v", names),
	}
	return &info, nil
}
