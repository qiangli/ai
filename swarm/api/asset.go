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
	return r.Roots.ResolvedRoots()
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
	// required primary working directory for the agents
	// default: $WORKSPACE env or system temp dir
	Workspace *Root `json:"workspace"`

	// optional current working directory to the root list
	// where ai is started
	// default: current working dir, $PWD or $(pwd)
	Cwd *Root `json:"cwd"`

	// optional system temporary directory to the root list
	// default: system temp dir
	Temp *Root `json:"temp"`

	// Additional allowed paths
	Dirs []*Root `json:"dirs"`

	//
	allowedDirs   []string
	resolvedRoots []*Root
	resolved      bool
}

func (r *Roots) ResolvedRoots() ([]*Root, error) {
	if r.resolved {
		return r.resolvedRoots, nil
	}
	if err := r.Resolve(); err != nil {
		return nil, err
	}
	return r.resolvedRoots, nil
}

func (r *Roots) Resolve() error {
	return r.resolveRoots()
}

func (r *Roots) resolveRoots() error {
	var roots []*Root
	for _, v := range r.Dirs {
		roots = append(roots, v)
	}

	// update path
	if r.Workspace == nil {
		return fmt.Errorf("workspace is required")
	}
	roots = append(roots, r.Workspace)
	if r.Workspace.Path == "" {
		ws := os.Getenv("WORKSPACE")
		if ws != "" {
			r.Workspace.Path = ws
		} else {
			r.Workspace.Path = os.TempDir()
		}
	}
	if r.Cwd != nil {
		roots = append(roots, r.Cwd)
		if r.Cwd.Path == "" {
			p, err := os.Getwd()
			if err != nil {
				return err
			}
			r.Cwd.Path = p
		}
	}
	if r.Temp != nil {
		roots = append(roots, r.Temp)
		if r.Temp.Path == "" {
			p := os.TempDir()
			r.Temp.Path = p
		}
	}

	// resolve relative path
	allPaths := make(map[string]struct{})
	for i, v := range roots {
		resolved, err := ResolvePath(v.Path)
		if err != nil {
			return err
		}
		roots[i].Path = resolved[0]
		for _, p := range resolved {
			allPaths[p] = struct{}{}
		}
	}

	var allowed []string
	for k := range allPaths {
		allowed = append(allowed, k)
	}

	r.resolved = true
	r.resolvedRoots = roots
	r.allowedDirs = allowed
	return nil
}

func (r *Roots) AllowedDirs() ([]string, error) {
	if r.resolved {
		return r.allowedDirs, nil
	}
	if err := r.Resolve(); err != nil {
		return nil, err
	}
	return r.allowedDirs, nil
}

type ResourceConfig struct {
	// file | web
	Type string `json:"type"`
	Base string `json:"base"`

	// web
	ApiKey string `json:"api_key"`
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
	//
	var roots = v.Roots
	if roots != nil {
		if err := roots.Resolve(); err != nil {
			return nil, err
		}
	}
	for i, a := range v.Assets {
		if a.Type == "file" {
			resolved, err := ResolvePath(a.Base)
			if err != nil {
				return nil, err
			}
			v.Assets[i].Base = resolved[0]
		}
	}
	// blob - not required for now
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
