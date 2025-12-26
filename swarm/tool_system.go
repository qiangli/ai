package swarm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
)

type toolSystem struct {
	kits map[any]api.ToolKit

	rte *api.ActionRTEnv
}

type KitKey struct {
	Type api.ToolType
	Kit  string
}

func NewKitKey(fnType api.ToolType, kit string) KitKey {
	return KitKey{
		Type: fnType,
		Kit:  kit,
	}
}

func NewToolSystem(rte *api.ActionRTEnv) (api.ToolSystem, error) {
	ts := &toolSystem{
		rte:  rte,
		kits: make(map[any]api.ToolKit),
	}

	kbPath := filepath.Join(rte.Base, "kb.json")
	if err := os.MkdirAll(kbPath, 0770); err != nil {
		return nil, err
	}

	// default by type
	ts.AddKit(api.ToolTypeFunc, atm.NewFuncKit(kbPath))
	ts.AddKit(api.ToolTypeWeb, atm.NewWebKit())
	ts.AddKit(api.ToolTypeSystem, atm.NewSystemKit())
	ts.AddKit(api.ToolTypeMcp, atm.NewMcpKit())
	// ts.AddKit(api.ToolTypeFaas, atm.NewFaasKit())

	return ts, nil
}

func (r *toolSystem) GetKit(key any) (api.ToolKit, error) {
	if key == nil {
		return nil, fmt.Errorf("kit key is nil")
	}
	tf, ok := key.(*api.ToolFunc)
	if ok {
		kk := NewKitKey(tf.Type, tf.Kit)
		if v, found := r.kits[kk]; found {
			return v, nil
		}
		if v, found := r.kits[tf.Type]; found {
			return v, nil
		}
		return nil, fmt.Errorf("toolkit %s (%s) not found", tf.Kit, tf.Type)
	}

	return nil, fmt.Errorf("toolkit %v not found", key)
}

func (r *toolSystem) AddKit(key any, kit api.ToolKit) {
	r.kits[key] = kit
}
