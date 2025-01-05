package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/tool"
)

type SqlAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewSqlAgent(cfg *llm.Config, role, content string) (*SqlAgent, error) {
	if role == "" {
		role = "system"
	}

	if content == "" {
		info, err := getDBInfo(cfg.DBConfig)
		if err != nil {
			return nil, err
		}
		systemMessage, err := resource.GetSqlSystemRoleContent(info)
		if err != nil {
			return nil, err
		}
		content = systemMessage
	}

	// Set up the tools
	cfg.Tools = tool.DBTools

	agent := SqlAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *SqlAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	var message = r.Message

	content, err := llm.Send(r.config, ctx, r.Role, message, input)
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