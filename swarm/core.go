package swarm

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
	resource "github.com/qiangli/ai/swarm/resource/agents"
)

type Swarm struct {
	Vars *api.Vars
}

func New(vars *api.Vars) *Swarm {
	return &Swarm{
		Vars: vars,
	}
}

var agentConfigMap = map[string][][]byte{}
var agentToolMap = map[string]*api.ToolFunc{}

func initToolAgents(app *api.AppConfig) error {
	resourceMap := resource.AgentCommandMap
	for k, v := range resourceMap {
		agentConfigMap[k] = [][]byte{resource.CommonData, v.Data}
	}

	// skip internal as tool - e.g launch
	for _, v := range resourceMap {
		if !app.Internal && v.Internal {
			continue
		}
		parts := strings.SplitN(v.Name, "/", 2)
		var service = parts[0]
		var toolName string
		if len(parts) == 2 {
			toolName = parts[1]
		}

		fn := &api.ToolFunc{
			Type:        ToolTypeAgent,
			Kit:         service,
			Name:        toolName,
			Description: v.Description,
		}
		agentToolMap[fn.ID()] = fn
	}
	return nil
}

func InitVars(app *api.AppConfig) (*api.Vars, error) {
	vars := api.NewVars()

	if err := initDefaultAgents(app); err != nil {
		return nil, err
	}
	if err := initToolAgents(app); err != nil {
		return nil, err
	}
	if err := initTools(app); err != nil {
		return nil, err
	}

	//
	vars.McpServerUrl = app.McpServerUrl

	//
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		return nil, err
	}

	//
	if app.Db != nil {
		vars.DBCred = &api.DBCred{
			Host:     app.Db.Host,
			Port:     app.Db.Port,
			Username: app.Db.Username,
			Password: app.Db.Password,
			DBName:   app.Db.DBName,
		}
	}

	//
	vars.Workspace = app.Workspace
	vars.Repo = app.Repo
	vars.Home = app.Home
	vars.Temp = app.Temp

	//
	vars.Arch = sysInfo.Arch
	vars.OS = sysInfo.OS
	vars.ShellInfo = sysInfo.ShellInfo
	vars.OSInfo = sysInfo.OSInfo
	vars.UserInfo = sysInfo.UserInfo

	//
	var modelMap = make(map[api.Level]*api.Model)
	modelMap[api.L1] = api.Level1(app.LLM)
	modelMap[api.L2] = api.Level2(app.LLM)
	modelMap[api.L3] = api.Level3(app.LLM)
	modelMap[api.LImage] = api.ImageModel(app.LLM)
	vars.Models = modelMap

	//
	// vars.AgentConfigMap = agentConfigMap

	vars.ResourceMap = resource.Prompts
	vars.TemplateFuncMap = tplFuncMap
	vars.AdviceMap = adviceMap
	vars.EntrypointMap = entrypointMap
	//
	vars.FuncRegistry = funcRegistry

	//
	toolMap := make(map[string]*api.ToolFunc)
	tools, err := listTools(app)
	if err != nil {
		return nil, err
	}
	for _, v := range tools {
		toolMap[v.ID()] = v
	}
	vars.ToolRegistry = toolMap

	return vars, nil
}

func (r *Swarm) Run(req *api.Request, resp *api.Response) error {
	for {
		agent, err := CreateAgent(r.Vars, req.Agent, req.Command, req.RawInput)
		if err != nil {
			return err
		}
		// agent.sw = r

		if agent.Entrypoint != nil {
			if err := agent.Entrypoint(r.Vars, &api.Agent{
				Name:    agent.Name,
				Display: agent.Display,
			}, req.RawInput); err != nil {
				return err
			}
		}

		timeout := TimeoutHandler(agent, time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)

		chain := NewChain(maxlog).Then(timeout)

		if err := chain.Serve(req, resp); err != nil {
			return err
		}

		// update the request
		result := resp.Result
		if result != nil && result.State == api.StateTransfer {
			req.Agent = result.NextAgent
			continue
		}
		return nil
	}
}

// MaxLogHandler returns a [Handler] that logs the request and response
func MaxLogHandler(n int) func(Handler) Handler {
	return func(next Handler) Handler {
		return &maxLogHandler{
			next: next,
			max:  n,
		}
	}
}

type maxLogHandler struct {
	next Handler
	max  int
}

func (h *maxLogHandler) Serve(r *api.Request, w *api.Response) error {

	log.Debugf("req: %+v\n", r)
	if r.Message != nil {
		log.Debugf("%s %s\n", r.Message.Role, clip(r.Message.Content, h.max))
	}

	err := h.next.Serve(r, w)

	log.Debugf("resp: %+v\n", w)
	if w.Messages != nil {
		for _, m := range w.Messages {
			log.Debugf("%s %s\n", m.Role, clip(m.Content, h.max))
		}
	}

	return err
}

// TimeoutHandler returns a [Handler] that runs h with the given time limit.
//
// The new Handler calls h.Serve to handle each request, but if a
// call runs for longer than its time limit, the handler responds with
// a timeout error.
// After such a timeout, writes by h to its [Response] will return
// [ErrHandlerTimeout].
func TimeoutHandler(next Handler, dt time.Duration, msg string) Handler {
	return &timeoutHandler{
		next:    next,
		content: msg,
		dt:      dt,
	}
}

// ErrHandlerTimeout is returned on [Response]
// in handlers which have timed out.
var ErrHandlerTimeout = errors.New("Handler timeout")

type timeoutHandler struct {
	next    Handler
	content string
	dt      time.Duration
}

func (h *timeoutHandler) Serve(r *api.Request, w *api.Response) error {
	ctx, cancelCtx := context.WithTimeout(r.Context(), h.dt)
	defer cancelCtx()

	r = r.WithContext(ctx)

	done := make(chan struct{})
	panicChan := make(chan any, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()

		h.next.Serve(r, w)

		close(done)
	}()

	select {
	case p := <-panicChan:
		return p.(error)
	case <-done:
		return nil
	case <-ctx.Done():
		w.Messages = []*api.Message{{Content: h.content}}
		w.Agent = nil
	}

	return ErrHandlerTimeout
}
