package atm

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sort"
	"strings"

	// "github.com/qiangli/ai/swarm/agent/api/entity"
	// "github.com/qiangli/ai/swarm/agent/internal/db"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
)

type FuncKit struct {
	User *api.User
}

// func (r *FuncKit) WhoAmI(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
// 	if r.user == nil {
// 		return "", fmt.Errorf("unknown user")
// 	}
// 	return fmt.Sprintf("Name: %s\nEmail: %s\nAvatar: %s\n", user.Name, user.Email, user.Avatar), nil
// }

// func (r *FuncKit) ListMembers(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
// 	var members []*entity.Member
// 	result := db.DB.Where("owner = ? AND status NOT IN ('suspended')", r.user.User.Email).Find(&members)
// 	if result.Error != nil {
// 		return "", result.Error
// 	}

// 	var list = []string{"Email,Name"}
// 	for _, v := range members {
// 		list = append(list, fmt.Sprintf("%s,%q", v.Email, v.Name))
// 	}
// 	return strings.Join(list, "\n"), nil
// }

// // list user who has shared agents with the current user
// func (r *FuncKit) ListOwners(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
// 	var owners []*entity.Member
// 	result := db.DB.Where("email = ? AND status NOT IN ('suspended')", r.user.User.Email).Find(&owners)
// 	if result.Error != nil {
// 		return "", result.Error
// 	}

// 	var list = []string{"Owner,Name"}
// 	for _, v := range owners {
// 		list = append(list, fmt.Sprintf("%s,%q", v.Owner, v.Name))
// 	}
// 	return strings.Join(list, "\n"), nil
// }

// func (r *FuncKit) ListModels(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
// 	var models []*entity.Model
// 	result := db.DB.Where("owner = ? AND status = 'active'", r.user.User.Email).Find(&models)
// 	if result.Error != nil {
// 		return "", result.Error
// 	}

// 	var list = []string{"Name,Display"}
// 	for _, v := range models {
// 		list = append(list, fmt.Sprintf("%s,%q", v.Name, v.Display))
// 	}
// 	return strings.Join(list, "\n"), nil
// }

// // TODO this is just a list of tool configs, not list of tools
// func (r *FuncKit) ListTools(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
// 	var tools []*entity.Tool
// 	result := db.DB.Where("owner = ?", r.user.User.Email).Find(&tools)
// 	if result.Error != nil {
// 		return "", result.Error
// 	}

// 	var list = []string{"Name,Display"}
// 	for _, v := range tools {
// 		list = append(list, fmt.Sprintf("%s,%q", v.Name, v.Display))
// 	}
// 	return strings.Join(list, "\n"), nil
// }

func (r *FuncKit) ListAgents(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
	var list []string

	dict, err := conf.ListAgents(r.User.Email)
	if err != nil {
		return "", err
	}

	for k, v := range dict {
		var desc []string
		for _, a := range v.Agents {
			desc = append(desc, a.Description)
		}
		list = append(list, fmt.Sprintf("%s: %s", k, strings.Join(desc, " ")))
	}
	sort.Strings(list)

	return fmt.Sprintf("Available agents:\n%s\n", strings.Join(list, "\n")), nil
}

func (r *FuncKit) AgentInfo(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return "", err
	}
	dict, err := conf.ListAgents(r.User.Email)
	if err != nil {
		return "", err
	}

	if v, ok := dict[agent]; ok {
		var desc []string
		for _, a := range v.Agents {
			desc = append(desc, a.Description)
		}
		return fmt.Sprintf("Agent: %s\nDescription: %s\n", v.Name, strings.Join(desc, " -- ")), nil
	}
	return "", fmt.Errorf("unknown agent: %s", agent)
}

func (r *FuncKit) AgentTransfer(_ context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		NextAgent: agent,
		State:     api.StateTransfer,
	}, nil
}

func (r *FuncKit) Call(ctx context.Context, vars *api.Vars, token api.SecretToken, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return CallKit(r, tf.Config.Kit, tf.Name, callArgs...)
}
