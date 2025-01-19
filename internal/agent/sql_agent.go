package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type SqlAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewSqlAgent(cfg *internal.AppConfig) (*SqlAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

	if role == "" {
		role = "system"
	}

	if prompt == "" {
		var err error
		info, err := getDBInfo(cfg.LLM.Sql.DBConfig)
		if err != nil {
			return nil, err
		}
		prompt, err = resource.GetSqlSystemRoleContent(info)
		if err != nil {
			return nil, err
		}
	}

	// Set up the tools
	cfg.LLM.Tools = llm.GetDBTools()

	agent := SqlAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *SqlAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	content, err := llm.Send(r.config.LLM, ctx, r.Role, r.Prompt, in.Input())
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "SQL",
		Content: content,
	}, nil
}

func getDBInfo(cfg *internal.DBConfig) (*resource.DBInfo, error) {
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
