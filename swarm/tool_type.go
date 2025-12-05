package swarm

import (
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

type toolSystem struct {
	workspace string
	user      *api.User
	assets    api.AssetManager
	secrets   api.SecretStore
	fs        vfs.FileSystem
	vs        vos.System
	//
	kits map[any]api.ToolKit
}

type KitKey struct {
	Type string
	Kit  string
}

func NewKitKey(fnType, kit string) KitKey {
	return KitKey{
		Type: fnType,
		Kit:  kit,
	}
}

func NewToolSystem(
	workspace string,
	user *api.User,
	secrets api.SecretStore,
	assets api.AssetManager,
	fs vfs.FileSystem,
	vs vos.System,
) api.ToolSystem {
	ts := &toolSystem{
		workspace: workspace,
		user:      user,
		secrets:   secrets,
		assets:    assets,
		fs:        fs,
		vs:        vs,
		//
		kits: make(map[any]api.ToolKit),
	}

	// web := atm.NewWebKit(secrets)
	// ts.AddKit(NewKitKey(api.ToolTypeFunc, "web"), web)

	// ts.AddKit(NewKitKey(api.ToolTypeFunc, "ddg"), web)
	// ts.AddKit(NewKitKey(api.ToolTypeFunc, "google"), web)
	// ts.AddKit(NewKitKey(api.ToolTypeFunc, "bing"), web)
	// ts.AddKit(NewKitKey(api.ToolTypeFunc, "brave"), web)
	// ts.AddKit(NewKitKey(api.ToolTypeFunc, "web"), web)

	// default by type
	ts.AddKit(api.ToolTypeFunc, atm.NewFuncKit(user, assets))
	ts.AddKit(api.ToolTypeWeb, atm.NewWebKit(secrets))
	ts.AddKit(api.ToolTypeSystem, atm.NewSystemKit(workspace, user, fs, vs, secrets))
	ts.AddKit(api.ToolTypeMcp, atm.NewMcpKit(secrets))
	ts.AddKit(api.ToolTypeFaas, atm.NewFaasKit(secrets))

	return ts
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
