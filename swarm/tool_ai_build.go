package swarm

import (
	"context"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
)

func (r *AIKit) BuildQuery(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
	// agent := args.Agent()
	// if agent == nil {
	// 	// return nil, fmt.Errorf("an instance of agent is required for buiding the query")
	// }
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	// convert user message into query if not set
	var query = args.Query()
	if !args.HasQuery() {
		msg := agent.Message
		if msg != "" {
			content, err := atm.CheckApplyTemplate(agent.Template, msg, args)
			if err != nil {
				return nil, err
			}
			query = content
		}
		//
		var userMsg = args.Message()
		query = api.Cat(query, userMsg, "\n")
		args.SetQuery(query)
	}

	return query, nil
}

func (r *AIKit) BuildPrompt(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
	// agent := args.Agent()
	// if agent == nil {
	// 	return nil, fmt.Errorf("an instance of agent is required for building the prompt")
	// }
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	var instructions []string

	add := func(in string) error {
		content, err := atm.CheckApplyTemplate(agent.Template, in, args)
		if err != nil {
			return err
		}

		// update instruction
		instructions = append(instructions, content)
		return nil
	}

	var addAll func(*api.Agent) error

	// inherit embedded agent instructions
	// merge all including the current agent
	addAll = func(a *api.Agent) error {
		for _, v := range a.Embed {
			if err := addAll(v); err != nil {
				return err
			}
		}
		in := a.Instruction
		if in != "" {
			if err := add(in); err != nil {
				return err
			}
		}
		return nil
	}

	// system role instructions
	var prompt = args.Prompt()
	if !args.HasPrompt() {
		if err := addAll(agent); err != nil {
			return nil, err
		}

		prompt = strings.Join(instructions, "\n")
		args.SetPrompt(prompt)
	}

	return prompt, nil
}

func (r *AIKit) BuildContext(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
	// agent := args.Agent()
	// if agent == nil {
	// 	return nil, fmt.Errorf("an instance of agent is required for building the context")
	// }
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	var history = args.History()

	if !args.HasHistory() {
		var c = agent.Context
		if c != "" {
			content, err := atm.CheckApplyTemplate(agent.Template, c, args)
			if err != nil {
				return nil, err
			}
			history = append(history, &api.Message{
				Content: content,
				Role:    api.RoleSystem,
			})

		} else {
			// load defaults
			if v, err := r.sw.History.Load(nil); err != nil {
				return nil, err
			} else {
				history = v
			}
		}
	}

	if len(history) > 0 {
		args.SetHistory(history)
	} else {
		delete(args, "history")
	}
	return history, nil
}
