package swarm

// import (
// 	"context"
// 	"fmt"
// 	"strings"

// 	"github.com/qiangli/ai/swarm/log"
// 	"github.com/qiangli/ai/swarm/api"
// )

// func callAgent(_ context.Context, vars *api.Vars, agent string, args map[string]any) (*api.Result, error) {
// 	log.Debugf("calling agent func %s\n", agent)

// 	// name/command
// 	var name, command string
// 	parts := strings.SplitN(agent, "/", 2)
// 	name = parts[0]
// 	if len(parts) > 1 {
// 		command = parts[1]
// 	}

// 	// _, ok := vars.AgentRegistry[name]
// 	// if !ok {
// 	// 	return nil, fmt.Errorf("failed: %s not found", name)
// 	// }

// 	req := &api.Request{
// 		Agent:    name,
// 		Command:  command,
// 		RawInput: vars.UserInput,
// 	}
// 	resp := &api.Response{}

// 	sw := New(vars)

// 	if err := sw.Run(req, resp); err != nil {
// 		return nil, err
// 	}

// 	log.Debugf("agent called:\n%+v\n", resp)

// 	if len(resp.Messages) == 0 {
// 		return nil, fmt.Errorf("empty result")
// 	}

// 	result := &api.Result{
// 		Value: resp.Messages[len(resp.Messages)-1].Content,
// 	}

// 	return result, nil
// }
