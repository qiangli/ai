package atm

import (
	"context"
	// "errors"
	"fmt"
	"maps"
	"net/url"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
)

// no-op tool that does nothing
func (r *SystemKit) Pass(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return "", nil
}

func (r *SystemKit) Exec(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	cmd, err := api.GetStrProp("command", args)
	if err != nil {
		return "", err
	}
	if len(cmd) == 0 {
		return "", fmt.Errorf("command is empty")
	}
	// command := argv[0]
	// rest := argv[1:]
	result, err := ExecCommand(ctx, vars.RTE.OS, vars, cmd, nil)
	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

func (r *SystemKit) Bash(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	// shell handles command/script if empty
	result, err := vars.RootAgent.Shell.Run(ctx, "", args)
	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

// template is required
func (r *SystemKit) Apply(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	tpl, err := api.GetStrProp("template", args)
	if err != nil {
		return "", err
	}
	if v, err := api.LoadURIContent(vars.RTE.Workspace, tpl); err != nil {
		return "", err
	} else {
		tpl = string(v)
	}

	var data = make(map[string]any)
	maps.Copy(data, vars.Global.GetAllEnvs())
	maps.Copy(data, args)

	return CheckApplyTemplate(vars.RootAgent.Template, tpl, data)
}

func (r *SystemKit) Parse(ctx context.Context, vars *api.Vars, name string, args map[string]any) (api.ArgMap, error) {
	result, err := conf.Parse(args["command"])
	if err != nil {
		return nil, err
	}
	return result, nil
}

// get default template based on format if template is not prvoided.
// tee content to destination specified by output param.
func (r *SystemKit) Format(ctx context.Context, vars *api.Vars, name string, args api.ArgMap) (string, error) {
	var tpl string
	tpl, _ = api.GetStrProp("template", args)
	if tpl != "" {
		if v, err := api.LoadURIContent(vars.RTE.Workspace, tpl); err != nil {
			return "", err
		} else {
			tpl = string(v)
		}
	}
	if tpl == "" {
		format, _ := api.GetStrProp("format", args)
		if format == "" {
			format = "markdown"
		}
		tpl = resource.FormatFile(format)
	}
	//
	output, _ := api.GetStrProp("output", args)

	var data = make(map[string]any)
	maps.Copy(data, vars.Global.GetAllEnvs())
	maps.Copy(data, args)

	txt, err := CheckApplyTemplate(vars.RootAgent.Template, tpl, data)
	if err != nil {
		return "", err
	}

	// tee output
	switch output {
	case "none", "":
		return txt, nil
	case "console":
		fmt.Printf("%s", txt)
		return txt, nil
	default:
		// uri
		uri, err := url.Parse(output)
		if err != nil {
			return "", err
		}
		if uri.Scheme != "file" {
			return "", fmt.Errorf("output scheme not supported: %s. write to file: or print on console", uri.Scheme)
		}
		err = vars.RTE.Workspace.WriteFile(uri.Path, []byte(txt))
		if err != nil {
			return "", err
		}
		return txt, nil
	}
}

// var ErrTimeout = errors.New("Action timeout")

func (r *SystemKit) Timeout(ctx context.Context, vars *api.Vars, name string, args api.ArgMap) (any, error) {
	// action := args.Actions()

	duration := args.GetDuration("duration")
	ctx, cancelCtx := context.WithTimeout(ctx, duration)
	defer cancelCtx()

	done := make(chan struct{})
	panicChan := make(chan any, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()

		// if err := h.next.Serve(nreq, resp); err != nil {
		// 	panicChan <- err
		// }

		close(done)
	}()

	select {
	case p := <-panicChan:
		return nil, p.(error)
	case <-done:
		return nil, nil
	case <-ctx.Done():
		// resp.Messages = []*api.Message{{Content: h.content}}
		// resp.Agent = nil
	}

	return nil, fmt.Errorf("action timedout")
}
