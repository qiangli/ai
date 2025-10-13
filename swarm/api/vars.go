package api

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

// global context
type Vars struct {
	Agent   string `json:"agent"`
	Message string `json:"message"`

	LogLevel LogLevel `json:"log_level"`

	ChatID     string `json:"chat_id"`
	MaxTurns   int    `json:"max_turns"`
	MaxTime    int    `json:"max_time"`
	New        *bool  `json:"new"`
	MaxHistory int    `json:"max_history"`
	MaxSpan    int    `json:"max_span"`
	Context    string `json:"context"`
	Format     string `json:"format"`
	Models     string `json:"models"`

	Unsafe    bool   `json:"unsafe"`
	Workspace string `json:"workspace"`

	DryRun        bool   `json:"-"`
	DryRunContent string `json:"-"`

	Extra map[string]string `json:"-"`

	// conversation history
	History []*Message `json:"-"`
}

func (v *Vars) Clone() *Vars {
	clone := &Vars{
		Agent:   v.Agent,
		Message: v.Message,
		//
		ChatID:     v.ChatID,
		New:        v.New,
		MaxHistory: v.MaxHistory,
		MaxSpan:    v.MaxSpan,
		Context:    v.Context,
		//
		MaxTurns: v.MaxTurns,
		MaxTime:  v.MaxTime,
		Models:   v.Models,
		//
		Format: v.Format,
		//
		Unsafe:    v.Unsafe,
		Workspace: v.Workspace,
		//
		LogLevel: v.LogLevel,
		//
		DryRun:        v.DryRun,
		DryRunContent: v.DryRunContent,
		//
		Extra:   make(map[string]string),
		History: make([]*Message, len(v.History)),
	}

	// Copy the Extra map
	for key, value := range v.Extra {
		clone.Extra[key] = value
	}

	// Copy the History slice
	copy(clone.History, v.History)

	return clone
}

func NewVars() *Vars {
	return &Vars{
		Extra: map[string]string{},
	}
}

func (r *Vars) IsTrace() bool {
	return r.LogLevel == Tracing
}

func (r *Vars) Get(key string) string {
	if r.Extra == nil {
		return ""
	}
	return r.Extra[key]
}

func (r *Vars) GetString(key string) string {
	if r.Extra == nil {
		return ""
	}
	v, ok := r.Extra[key]
	if !ok {
		return ""
	}
	return v
}
