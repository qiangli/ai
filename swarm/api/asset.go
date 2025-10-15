package api

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
)

// TODO
type ResourceConfig struct {
	Base  string `json:"base"`
	Token string `json:"token"`
}

func LoadResourceConfig(conf string) (*ResourceConfig, error) {
	var v ResourceConfig
	d, err := os.ReadFile(conf)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(d, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

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
	SearchAgent(owner, pack string) (*Record, error)
	RetrieveTool(owner, kit string) (*Record, error)
	ListTools(owner string) ([]*Record, error)
	RetrieveModel(owner, alias string) (*Record, error)
	ListModels(owner string) ([]*Record, error)
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

	SearchAgent(owner, pack string) (*Record, error)
	ListAgent(owner string) (map[string]*AgentsConfig, error)
	FindAgent(owner, pack string) (*AgentsConfig, error)
	ListToolkit(owner string) (map[string]*ToolsConfig, error)
	FindToolkit(owner string, kit string) (*ToolsConfig, error)
	ListModels(owner string) (map[string]*ModelsConfig, error)
	FindModels(owner string, alias string) (*ModelsConfig, error)
}
