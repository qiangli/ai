package api

import (
	"strings"
)

type AppConfig struct {
	Version string

	ConfigFile string

	AgentResource *AgentResource

	// AgentCreator AgentCreator
	// AgentHandler AgentHandler
	// ToolCaller   ToolCaller

	// ToolSystemCommands []string
	SystemTools []*ToolFunc

	Agent string
	// Command string
	Args []string

	// --message takes precedence, skip stdin
	// command line arguments
	Message string

	// editor binary and args. e.g vim [options]
	Editor string

	Clipin   bool
	ClipWait bool

	Clipout    bool
	ClipAppend bool

	IsPiped bool
	Stdin   bool

	Files []string

	// Treated as file
	Screenshot bool

	// Treated as input text
	Voice bool

	// Output format: raw or markdown
	Format string

	// output file for saving response
	Output string

	Me *User

	//
	Template string

	// conversation history
	New        bool
	MaxHistory int
	MaxSpan    int

	// chat id to continue the conersation
	// <config_base>/chat/<id>.json
	ChatID string

	//<config_base>/chat/<id>/*.json
	// History []*Message
	History MemStore

	Models string

	// Log string

	// TODO change to log level?
	Trace bool
	Debug bool
	Quiet bool

	// Internal bool

	DenyList  []string
	AllowList []string
	Unsafe    bool

	//
	Base string

	Workspace string
	Home      string
	Temp      string

	Interactive bool
	Editing     bool
	Shell       string

	Watch     bool
	ClipWatch bool

	MaxTime  int
	MaxTurns int

	//
	Stdout string
	Stderr string

	// dry run
	DryRun        bool
	DryRunContent string

	// experimental
	// Env map[string]string
}

// Clone is a shallow copy of member fields of the configration
func (cfg *AppConfig) Clone() *AppConfig {
	return &AppConfig{
		Version:    cfg.Version,
		ConfigFile: cfg.ConfigFile,
		//
		AgentResource: cfg.AgentResource,
		// AgentCreator:  cfg.AgentCreator,
		// AgentHandler:  cfg.AgentHandler,
		//
		Agent: cfg.Agent,
		// Command: cfg.Command,
		// Args:       append([]string(nil), cfg.Args...),
		Args:       cfg.Args,
		Message:    cfg.Message,
		Editor:     cfg.Editor,
		Clipin:     cfg.Clipin,
		ClipWait:   cfg.ClipWait,
		Clipout:    cfg.Clipout,
		ClipAppend: cfg.ClipAppend,
		IsPiped:    cfg.IsPiped,
		Stdin:      cfg.Stdin,
		// Files:         append([]string(nil), cfg.Files...),
		Files:      cfg.Files,
		Screenshot: cfg.Screenshot,
		Voice:      cfg.Voice,
		Format:     cfg.Format,
		Output:     cfg.Output,
		Me:         cfg.Me,
		Template:   cfg.Template,
		New:        cfg.New,
		ChatID:     cfg.ChatID,
		MaxHistory: cfg.MaxHistory,
		MaxSpan:    cfg.MaxSpan,
		// History:    cfg.History,
		Models: cfg.Models,
		// Log:    cfg.Log,
		Debug: cfg.Debug,
		Quiet: cfg.Quiet,

		// DenyList:  append([]string(nil), cfg.DenyList...),
		// AllowList: append([]string(nil), cfg.AllowList...),
		DenyList:    cfg.DenyList,
		AllowList:   cfg.AllowList,
		Unsafe:      cfg.Unsafe,
		Base:        cfg.Base,
		Workspace:   cfg.Workspace,
		Home:        cfg.Home,
		Temp:        cfg.Temp,
		Interactive: cfg.Interactive,
		Editing:     cfg.Editing,
		Shell:       cfg.Shell,
		Watch:       cfg.Watch,
		ClipWatch:   cfg.ClipWatch,
		// Hub:         cfg.Hub,
		MaxTime:  cfg.MaxTime,
		MaxTurns: cfg.MaxTurns,
		Stdout:   cfg.Stdout,
		Stderr:   cfg.Stderr,
		//
		DryRun:        cfg.DryRun,
		DryRunContent: cfg.DryRunContent,
		//
		// Env: cfg.Env,
	}
}

func (r *AppConfig) IsStdin() bool {
	return r.Stdin || r.IsPiped
}

func (r *AppConfig) IsClipin() bool {
	return r.Clipin
}

func (r *AppConfig) IsMedia() bool {
	return r.Screenshot || r.Voice
}

func (r *AppConfig) IsSpecial() bool {
	return r.IsStdin() || r.IsClipin() || r.IsMedia()
}

func (r *AppConfig) HasInput() bool {
	return r.Message != "" || len(r.Files) > 0 || len(r.Args) > 0
}

func (r *AppConfig) GetQuery() string {
	if r.Message != "" {
		return r.Message
	}
	return strings.Join(r.Args, " ")
}
