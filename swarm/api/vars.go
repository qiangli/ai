package api

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

// global context
type Vars struct {
	// Config *AppConfig `json:"config"`
	LogLevel   LogLevel
	ChatID     string
	MaxTurns   int
	MaxTime    int
	New        *bool
	MaxHistory int
	MaxSpan    int
	Context    string
	Message    string
	Format     string
	Models     string

	Unsafe bool

	DryRun        bool
	DryRunContent string

	//
	Workspace string `json:"workspace"`

	//
	Extra map[string]string `json:"extra"`

	// conversation history
	History []*Message
}

func (v *Vars) Clone() *Vars {
	clone := &Vars{
		// Config:    v.Config,
		LogLevel:   v.LogLevel,
		ChatID:     v.ChatID,
		New:        v.New,
		MaxTurns:   v.MaxTurns,
		MaxTime:    v.MaxTime,
		MaxHistory: v.MaxHistory,
		MaxSpan:    v.MaxSpan,
		Context:    v.Context,
		Message:    v.Message,
		Format:     v.Format,
		Models:     v.Models,
		//
		Unsafe: v.Unsafe,
		//
		DryRun:        v.DryRun,
		DryRunContent: v.DryRunContent,
		//
		Workspace: v.Workspace,
		Extra:     make(map[string]string),
		History:   make([]*Message, len(v.History)),
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
