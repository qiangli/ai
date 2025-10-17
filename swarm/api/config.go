package api

import (
	"strings"
)

type LogLevel int

func (r LogLevel) String() string {
	return LogLevelToString(r)
}

const (
	Quiet LogLevel = iota + 1
	Informative
	Verbose
	Tracing
)

func LogLevelToString(level LogLevel) string {
	switch level {
	case Quiet:
		return "Quiet"
	case Informative:
		return "Informative"
	case Verbose:
		return "Verbose"
	case Tracing:
		return "Tracing"
	default:
		return ""
	}
}

func ToLogLevel(level string) LogLevel {

	switch strings.ToLower(level) {
	case "quiet":
		return Quiet
	case "info", "informative":
		return Informative
	case "debug", "verbose":
		return Verbose
	case "trace", "tracing":
		return Tracing
	default:
		return 0
	}
}

type AppConfig struct {
	Version string

	ConfigFile string

	// // rename WebResource?
	// AgentResource *AgentResource

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

	// IsPiped bool
	Stdin bool

	// Output format: raw or markdown
	Format string

	// output file for saving response
	Output string

	// user
	User *User

	// chat id to continue the conersation
	// <config_base>/chat/<id>.json
	ChatID string

	// conversation history
	New        *bool
	MaxHistory int
	MaxSpan    int

	// history @agent
	Context string

	Models string

	//
	LogLevel string

	Unsafe bool

	//
	Base string

	Workspace string

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
}

// Clone is a shallow copy of member fields of the configration
func (cfg *AppConfig) Clone() *AppConfig {
	return &AppConfig{
		Version:    cfg.Version,
		ConfigFile: cfg.ConfigFile,
		//
		// AgentResource: cfg.AgentResource,
		//
		Agent:  cfg.Agent,
		Models: cfg.Models,
		//
		Args:       cfg.Args,
		Message:    cfg.Message,
		Editor:     cfg.Editor,
		Clipin:     cfg.Clipin,
		ClipWait:   cfg.ClipWait,
		Clipout:    cfg.Clipout,
		ClipAppend: cfg.ClipAppend,
		// IsPiped:    cfg.IsPiped,
		Stdin: cfg.Stdin,
		//
		Format: cfg.Format,
		Output: cfg.Output,
		//
		ChatID:     cfg.ChatID,
		New:        cfg.New,
		MaxHistory: cfg.MaxHistory,
		MaxSpan:    cfg.MaxSpan,
		Context:    cfg.Context,
		//
		LogLevel: cfg.LogLevel,
		//
		Unsafe:      cfg.Unsafe,
		Base:        cfg.Base,
		Workspace:   cfg.Workspace,
		Interactive: cfg.Interactive,
		Editing:     cfg.Editing,
		Shell:       cfg.Shell,
		Watch:       cfg.Watch,
		ClipWatch:   cfg.ClipWatch,
		MaxTime:     cfg.MaxTime,
		MaxTurns:    cfg.MaxTurns,
		Stdout:      cfg.Stdout,
		Stderr:      cfg.Stderr,
		//
		DryRun:        cfg.DryRun,
		DryRunContent: cfg.DryRunContent,
	}
}

func (cfg *AppConfig) IsNew() bool {
	return cfg.New != nil && *cfg.New
}

func (cfg *AppConfig) IsQuiet() bool {
	return ToLogLevel(cfg.LogLevel) == Quiet
}

func (cfg *AppConfig) IsInformative() bool {
	return ToLogLevel(cfg.LogLevel) == Informative
}

func (cfg *AppConfig) IsVerbose() bool {
	return ToLogLevel(cfg.LogLevel) == Verbose
}

func (cfg *AppConfig) IsTracing() bool {
	return ToLogLevel(cfg.LogLevel) == Tracing
}

func (r *AppConfig) IsStdin() bool {
	return r.Stdin
}

func (r *AppConfig) IsClipin() bool {
	return r.Clipin
}

func (r *AppConfig) IsSpecial() bool {
	return r.IsStdin() || r.IsClipin()
}

func (r *AppConfig) HasInput() bool {
	return r.Message != "" || len(r.Args) > 0
}
