package swarm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

type AppConfig = internal.AppConfig
type Swarm struct {
	AppConfig *AppConfig

	History []*Message
	Vars    *Vars
	Stream  bool

	// RawInput *UserInput

	//
	Config *AgentsConfig
	// Create AgentFunc

	//
	DryRun        bool
	DryRunContent string

	// starter
	Agent string

	AgentConfigMap map[string][][]byte

	ResourceMap     map[string]string     `yaml:"-"`
	AdviceMap       map[string]Advice     `yaml:"-"`
	EntrypointMap   map[string]Entrypoint `yaml:"-"`
	TemplateFuncMap template.FuncMap      `yaml:"-"`

	FuncRegistry map[string]Function
}

func NewSwarm(app *AppConfig) (*Swarm, error) {
	sw := &Swarm{
		Vars:      NewVars(),
		History:   []*Message{},
		Stream:    true,
		AppConfig: app,
	}

	if err := sw.initVars(); err != nil {
		return nil, err
	}

	return sw, nil
}

// loadAgentsConfig loads the agent configuration from the provided YAML data.
func (r *Swarm) loadAgentsConfig(data [][]byte) error {
	merged := &AgentsConfig{}

	for _, v := range data {
		cfg := &AgentsConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return err
		}
	}

	merged.ResourceMap = r.ResourceMap
	merged.TemplateFuncMap = r.TemplateFuncMap

	// TODO per agent?
	merged.AdviceMap = r.AdviceMap
	merged.EntrypointMap = r.EntrypointMap

	r.Config = merged
	// r.Create = AgentCreator(merged)

	return nil
}

func (r *Swarm) initVars() error {
	app := r.AppConfig

	//
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		return err
	}

	//
	// r.Vars.Role = app.Role
	// r.Vars.Prompt = app.Prompt

	if app.Db != nil {
		r.Vars.DBCred = &DBCred{
			Host:     app.Db.Host,
			Port:     app.Db.Port,
			Username: app.Db.Username,
			Password: app.Db.Password,
			DBName:   app.Db.DBName,
		}
	}
	//
	r.Vars.Arch = sysInfo.Arch
	r.Vars.OS = sysInfo.OS
	r.Vars.ShellInfo = sysInfo.ShellInfo
	r.Vars.OSInfo = sysInfo.OSInfo
	r.Vars.UserInfo = sysInfo.UserInfo
	r.Vars.WorkDir = sysInfo.WorkDir

	return nil
}

func (r *Swarm) Load(name string, input *UserInput) error {
	// check if the agent is already loaded
	if r.Config != nil && len(r.Config.Agents) > 0 {
		for _, a := range r.Config.Agents {
			if a.Name == name {
				return nil
			}
		}
	}

	data, ok := r.AgentConfigMap[name]
	if !ok {
		return internal.NewUserInputError("not supported yet: " + name)
	}
	err := r.loadAgentsConfig(data)
	if err != nil {
		return err
	}
	r.Agent = name

	app := r.AppConfig
	config := r.Config

	var modelMap = make(map[string]*api.Model)
	for _, m := range config.Models {
		if m.External {
			switch m.Name {
			case "L1":
				modelMap["L1"] = internal.Level1(app.LLM)
			case "L2":
				modelMap["L2"] = internal.Level2(app.LLM)
			case "L3":
				modelMap["L3"] = internal.Level3(app.LLM)
			case "Image":
				modelMap["Image"] = internal.ImageModel(app.LLM)
			}
		} else {
			modelMap[m.Name] = &api.Model{
				Type:    api.ModelType(m.Type),
				Name:    m.Model,
				BaseUrl: m.BaseUrl,
				ApiKey:  m.ApiKey,
			}
		}
	}

	var functionMap = make(map[string]*ToolFunc)
	for _, v := range config.Functions {
		functionMap[v.Name] = &ToolFunc{
			Name:        v.Name,
			Description: v.Description,
			Parameters:  v.Parameters,
		}
	}

	//
	r.Vars.Models = modelMap
	r.Vars.Functions = functionMap
	r.Vars.FuncRegistry = r.FuncRegistry

	return nil
}

func (r *Swarm) Run(req *Request, resp *Response) error {
	// "resource:" prefix is used to refer to a resource
	// "vars:" prefix is used to refer to a variable
	apply := func(s string, vars *Vars) (string, error) {
		if strings.HasPrefix(s, "resource:") {
			v, ok := r.Config.ResourceMap[s[9:]]
			if !ok {
				return "", fmt.Errorf("no such resource: %s", s[9:])
			}
			return applyTemplate(v, vars, r.Config.TemplateFuncMap)
		}
		if strings.HasPrefix(s, "vars:") {
			v := vars.GetString(s[5:])
			return v, nil
		}
		return s, nil
	}

	for {
		agent, err := r.Create(req.Agent, req.RawInput)
		if err != nil {
			return err
		}
		agent.sw = r

		if agent.Entrypoint != nil {
			if err := agent.Entrypoint(r.Vars, agent, req.RawInput); err != nil {
				return err
			}
		}

		// update the request instruction
		content, err := apply(agent.Instruction, r.Vars)
		if err != nil {
			return err
		}
		agent.Instruction = content

		timeout := TimeoutHandler(agent, time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)

		chain := New(maxlog).Then(timeout)

		if err := chain.Serve(req, resp); err != nil {
			return err
		}

		// update the request
		if resp.Transfer {
			req.Agent = resp.NextAgent
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

func (h *maxLogHandler) Serve(r *Request, w *Response) error {

	log.Debugf("req: %+v\n", r)
	if r.Message != nil {
		log.Printf("%s %s\n", r.Message.Role, clip(r.Message.Content, h.max))
	}

	err := h.next.Serve(r, w)

	log.Debugf("resp: %+v\n", w)
	if w.Messages != nil {
		for _, m := range w.Messages {
			log.Printf("%s %s\n", m.Role, clip(m.Content, h.max))
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

func (h *timeoutHandler) Serve(r *Request, w *Response) error {
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
		w.Messages = []*Message{{Content: h.content}}
		w.Agent = nil
	}

	return ErrHandlerTimeout
}
