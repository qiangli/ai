package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type DHNTConfig struct {
	// base where this config loaded from. default $HOME/.ai/
	// filename: dhnt.json
	Base string `json:"-"`

	// additional work spaces
	Roots *Roots `json:"roots"`

	Blob   *ResourceConfig   `json:"blob"`
	Assets []*ResourceConfig `json:"assets"`
}

// Return root paths
func (r *DHNTConfig) GetRoots() ([]*Root, error) {
	if r.Roots == nil {
		return nil, nil
	}
	return r.Roots.ResolveRoots()
}

// https://modelcontextprotocol.io/specification/2025-06-18/client/roots
// https://www.rfc-editor.org/rfc/rfc3986
// https://en.wikipedia.org/wiki/Uniform_Resource_Identifier
// URI = scheme ":" ["//" authority] path ["?" query] ["#" fragment]
type Root struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Scheme      string `json:"scheme"`
	Path string `json:"path"`
}

type Roots struct {
	// primary working directory for the agents
	Workspace *Root `json:"workspace"`

	// // Add user home to the root list
	// Home bool `json:"home"`

	// Add the current working directory to the root list
	// where ai is started
	Cwd *Root `json:"cwd"`

	// Add system temporary directory to the root list
	Temp *Root `json:"temp"`

	// Additional paths
	Dirs []*Root `json:"dirs"`
}

// // Returns temp directory if not set
// func (r *Roots) Workspace() string {
// 	if r.root == nil || r.root.Path == "" {
// 		return os.TempDir()
// 	}
// 	return r.root.Path
// }

func (r *Roots) ResolveRoots() ([]*Root, error) {
	var ps []*Root
	for _, v := range r.Dirs {
		ps = append(ps, v)
	}

	// update path
	if r.Workspace == nil {
		return nil, fmt.Errorf("workspace is required")
	}
	ps = append(ps, r.Workspace)
	if r.Workspace.Path == "" {
		r.Workspace.Path = os.TempDir()
	}
	if r.Cwd != nil {
		ps = append(ps, r.Cwd)
		if r.Cwd.Path == "" {
			p, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			r.Cwd.Path = p
		}
	}
	if r.Temp != nil {
		ps = append(ps, r.Temp)
		if r.Temp.Path == "" {
			p := os.TempDir()
			r.Temp.Path = p
		}
	}
	return ps, nil
}

// convenience helper for collecting all accessible paths
// including symlink of the original path
func (r *Roots) AllowedDirs() ([]string, error) {
	roots, err := r.ResolveRoots()
	if err != nil {
		return nil, err
	}
	var ps []string
	for _, v := range roots {
		ps = append(ps, v.Path)
	}
	return resolvePaths(ps)
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
	// SearchAgent(owner, pack string) (*Record, error)
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

	// SearchAgent(owner, pack string) (*Record, error)
	ListAgent(owner string) (map[string]*AppConfig, error)
	FindAgent(owner, pack string) (*AppConfig, error)
	ListToolkit(owner string) (map[string]*AppConfig, error)
	FindToolkit(owner string, kit string) (*AppConfig, error)
	ListModels(owner string) (map[string]*AppConfig, error)
	FindModels(owner string, alias string) (*AppConfig, error)
}
