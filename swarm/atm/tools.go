package atm

import (
	"fmt"

	"github.com/qiangli/ai/swarm/api"
)

type ContextKey string

const ModelsContextKey = "eval-models"

type toolSystem struct {
	user *api.User
	kits map[string]api.ToolKit
}

func NewToolSystem(user *api.User) api.ToolSystem {
	return &toolSystem{
		user: user,
		kits: make(map[string]api.ToolKit),
	}
}

func (r *toolSystem) GetKit(key string) (api.ToolKit, error) {
	if v, ok := r.kits[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("toolkit %q not found", key)
}

func (r *toolSystem) AddKit(key string, kit api.ToolKit) {
	r.kits[key] = kit
}
