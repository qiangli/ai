package swarm

import (
	"context"
	"errors"
	"text/template"
	"time"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm/vfs"
	"github.com/qiangli/ai/internal/util"
)

type AppConfig = internal.AppConfig
type Swarm struct {
	AppConfig *AppConfig

	History []*Message
	Vars    *Vars
	Stream  bool

	//
	Config *AgentsConfig

	//
	DryRun        bool
	DryRunContent string

	// McpServerTool *McpServerTool

	// ToolMap      map[string]*ToolFunc
	// FuncRegistry map[string]Function

	// map of agent name to the agent configuration data.
	AgentConfigMap map[string][][]byte

	ResourceMap     map[string]string
	AdviceMap       map[string]Advice
	EntrypointMap   map[string]Entrypoint
	TemplateFuncMap template.FuncMap

	//
	// fs vfs.FileSystem
}

func NewSwarm(app *AppConfig) (*Swarm, error) {
	sw := &Swarm{
		AppConfig: app,
		History:   []*Message{},
		Stream:    true,
	}

	if err := sw.initVars(); err != nil {
		return nil, err
	}

	return sw, nil
}

// loadAgentsConfig loads the agent configuration from the provided YAML data.
func loadAgentsConfig(data [][]byte) (*AgentsConfig, error) {
	merged := &AgentsConfig{}

	for _, v := range data {
		cfg := &AgentsConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func (r *Swarm) loadAgentsConfig(data [][]byte) error {
	merged, err := loadAgentsConfig(data)
	if err != nil {
		return err
	}
	merged.ResourceMap = r.ResourceMap
	merged.TemplateFuncMap = r.TemplateFuncMap

	// TODO per agent?
	merged.AdviceMap = r.AdviceMap
	merged.EntrypointMap = r.EntrypointMap

	r.Config = merged

	return nil
}

func (r *Swarm) initVars() error {
	vars := NewVars()

	app := r.AppConfig

	//
	server := NewMcpServerTool(app.McpServerUrl)
	vars.McpServerTool = server

	//
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		return err
	}

	//
	if r.AppConfig.Db != nil {
		vars.DBCred = &DBCred{
			Host:     r.AppConfig.Db.Host,
			Port:     r.AppConfig.Db.Port,
			Username: r.AppConfig.Db.Username,
			Password: r.AppConfig.Db.Password,
			DBName:   r.AppConfig.Db.DBName,
		}
	}

	//
	vars.Arch = sysInfo.Arch
	vars.OS = sysInfo.OS
	vars.ShellInfo = sysInfo.ShellInfo
	vars.OSInfo = sysInfo.OSInfo
	vars.UserInfo = sysInfo.UserInfo
	vars.WorkDir = sysInfo.WorkDir
	// vars.Roots = r.AppConfig.Roots

	//
	var modelMap = make(map[api.Level]*api.Model)
	modelMap[api.L1] = api.Level1(app.LLM)
	modelMap[api.L2] = api.Level2(app.LLM)
	modelMap[api.L3] = api.Level3(app.LLM)
	modelMap[api.LImage] = api.ImageModel(app.LLM)
	vars.Models = modelMap

	//
	fs, err := vfs.NewVFS()
	if err != nil {
		return err
	}
	vars.FS = fs

	r.Vars = vars
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

	// override
	app := r.AppConfig
	config := r.Config

	var modelMap = make(map[api.Level]*api.Model)
	for _, m := range config.Models {
		if m.External {
			switch m.Name {
			case "L1":
				modelMap[api.L1] = api.Level1(app.LLM)
			case "L2":
				modelMap[api.L2] = api.Level2(app.LLM)
			case "L3":
				modelMap[api.L3] = api.Level3(app.LLM)
			case "Image":
				modelMap[api.LImage] = api.ImageModel(app.LLM)
			}
		} else {
			l := toModelLevel(m.Name)
			modelMap[l] = &api.Model{
				Type:    api.ModelType(m.Type),
				Name:    m.Model,
				BaseUrl: m.BaseUrl,
				ApiKey:  m.ApiKey,
			}
		}
	}
	r.Vars.Models = modelMap

	return nil
}

func (r *Swarm) Run(req *Request, resp *Response) error {
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

		timeout := TimeoutHandler(agent, time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)

		chain := New(maxlog).Then(timeout)

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

func (h *maxLogHandler) Serve(r *Request, w *Response) error {

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
