package api

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type DHNTConfig struct {
	Base string `json:"-"`

	Roots Roots `json:"roots"`

	Blob   *ResourceConfig   `json:"blob"`
	Assets []*ResourceConfig `json:"assets"`
}

type ResourceConfig struct {
	Type  string `json:"type"`
	Base  string `json:"base"`
	Token string `json:"token"`
}

func LoadDHNTConfig(conf string) (*DHNTConfig, error) {
	var v DHNTConfig
	d, err := os.ReadFile(conf)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(d, &v); err != nil {
		return nil, err
	}
	v.Base = filepath.Dir(conf)
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
