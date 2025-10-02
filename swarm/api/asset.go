package api

import (
	"github.com/google/uuid"
)

type Record struct {
	ID uuid.UUID

	Owner   string
	Name    string
	Display string
	Content string
}

// agent/tool/model methods
type ATMSupport interface {
	RetrieveAgent(owner, pack string) (*Record, error)
	ListAgents(owner string) ([]*Record, error)
	SearchAgent(user, owner, pack string) (*Record, error)
	RetrievTool(owner, kit string) (*Record, error)
	RetrieveModel(owner, alias string) (*Record, error)
}

type AssetStore interface {
	ReadDir(name string) ([]DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Resolve(parent string, name string) string
	Search(query string) ([]byte, error)
}

type AssetManager interface {
	// GetStore(key string) (AssetStore, error)
	AddStore(store AssetStore)

	// @<[partial owner:]agent>
	// owner, agentName := splitOwnerAgent(req.Agent)
	// agent: [pack/]sub
	// pack, sub := split2(agentName, "/", "")
	SearchAgent(owner, pack string) (*AgentsConfig, error)
	// FindAgent(ownr, pack string) (*AgentsConfig, error)

	ListAgent(owner string) (map[string]*AgentsConfig, error)

	FindToolkit(owner string, kit string) (*ToolsConfig, error)

	FindModels(owner string, alias string) (*ModelsConfig, error)
}
