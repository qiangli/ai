package api

import (
	"github.com/google/uuid"
)

type AssetStore any

type Record struct {
	ID uuid.UUID

	Owner   string
	Name    string
	Display string
	Content string

	// source
	Store AssetStore
}

// agent/tool/model methods
type ATMSupport interface {
	AssetStore
	RetrieveAgent(owner, pack string) (*Record, error)
	ListAgents(owner string) ([]*Record, error)
	SearchAgent(user, owner, pack string) (*Record, error)
	RetrieveTool(owner, kit string) (*Record, error)
	RetrieveModel(owner, alias string) (*Record, error)
}

type AssetFS interface {
	AssetStore
	ReadDir(name string) ([]DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Resolve(parent string, name string) string
}

type AssetManager interface {
	// GetStore(key string) (AssetStore, error)
	AddStore(store AssetStore)

	// @<[partial owner:]agent>
	// owner, agentName := splitOwnerAgent(req.Agent)
	// agent: [pack/]sub
	// pack, sub := split2(agentName, "/", "")
	SearchAgent(owner, pack string) (*Record, error)
	ListAgent(owner string) (map[string]*AgentsConfig, error)
	FindAgent(owner, pack string) (*AgentsConfig, error)
	FindToolkit(owner string, kit string) (*ToolsConfig, error)
	FindModels(owner string, alias string) (*ModelsConfig, error)
}
