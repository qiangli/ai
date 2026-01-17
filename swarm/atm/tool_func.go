package atm

import (
	"context"
	"fmt"
	// "path"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/tool/memory"
)

type FuncKit struct {
	// rte *api.ActionRTEnv
	kb *memory.KnowledgeBase
}

func NewFuncKit(kbPath string) *FuncKit {
	// kbPath := path.Join(rte.Base, "kb")
	return &FuncKit{
		// rte: rte,
		kb: memory.NewKnowlegeBase(kbPath),
	}
}

func (r *FuncKit) Call(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	if tf.Body == nil {
		return r.builtin(ctx, vars, parent, tf, args)
	}

	// template is applied before running application code scripts
	// script is retured as is for text/*
	mime := strings.ToLower(tf.Body.MimeType)
	switch mime {
	case "application/x-sh", "text/x-shellscript", "application/x-shellscript", "bash", "sh":
		return r.ExecScript(ctx, vars, parent, tf, args)
	case "application/yaml", "yaml", "yml":
		return r.ExecScript(ctx, vars, parent, tf, args)
	case "application/x-go", "go", "golang":
		return r.ExecScript(ctx, vars, parent, tf, args)
	case "text/x-go-template", "template", "tpl":
		return r.ExecScript(ctx, vars, parent, tf, args)
	case "text/uri-list", "uri":
		// TODO
		return nil, fmt.Errorf("mime type not supported: %s", mime)
	case "text/javascript", "js", "javascript", "ecmascript":
		return r.ExecScript(ctx, vars, parent, tf, args)
	case "py", "python":
		return r.DO(ctx, vars, parent, tf, args)
	default:
		if strings.HasPrefix(mime, "text/") {
			return tf.Body.Script, nil
		}
	}
	return nil, fmt.Errorf("mime type not supported: %s", mime)
}

func (r *FuncKit) builtin(ctx context.Context, vars *api.Vars, _ *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return CallKit(r, tf.Kit, tf.Name, callArgs...)
}
