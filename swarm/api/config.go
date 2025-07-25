package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qiangli/ai/swarm/api/model"
)

type AppConfig struct {
	Version string

	ConfigFile string

	LLM *LLMConfig

	TTS *TTSConfig

	Git    *GitConfig
	DBCred *DBCred

	// system
	Role   string
	Prompt string

	Agent   string
	Command string
	Args    []string

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

	// MCP server
	McpServerRoot string
	McpServers    map[string]*McpServerConfig

	// Output format: raw or markdown
	Format string

	// output file for saving response
	Output string

	Me string

	//
	Template string

	// conversation history
	New        bool
	MaxHistory int
	MaxSpan    int

	History []*Message
	Models  string

	Log      string
	Debug    bool
	Quiet    bool
	Internal bool

	DenyList  []string
	AllowList []string
	Unsafe    bool

	//
	Base string

	Workspace string
	// Repo      string
	Home string
	Temp string

	Interactive bool
	Editing     bool
	Shell       string

	Watch     bool
	ClipWatch bool

	Hub *HubConfig

	MaxTime  int
	MaxTurns int

	//
	Stdout string
	Stderr string
}

// Clone is a shallow copy of member fields of the configration
func (cfg *AppConfig) Clone() *AppConfig {
	if cfg.LLM == nil {
		cfg.LLM = &LLMConfig{}
	}
	if cfg.TTS == nil {
		cfg.TTS = &TTSConfig{}
	}
	var llm = cfg.LLM.Clone()
	var tts = cfg.TTS.Clone()
	return &AppConfig{
		Version:       cfg.Version,
		ConfigFile:    cfg.ConfigFile,
		LLM:           llm,
		TTS:           tts,
		Git:           cfg.Git,
		DBCred:        cfg.DBCred,
		Role:          cfg.Role,
		Prompt:        cfg.Prompt,
		Agent:         cfg.Agent,
		Command:       cfg.Command,
		Args:          append([]string(nil), cfg.Args...),
		Message:       cfg.Message,
		Editor:        cfg.Editor,
		Clipin:        cfg.Clipin,
		ClipWait:      cfg.ClipWait,
		Clipout:       cfg.Clipout,
		ClipAppend:    cfg.ClipAppend,
		IsPiped:       cfg.IsPiped,
		Stdin:         cfg.Stdin,
		Files:         append([]string(nil), cfg.Files...),
		Screenshot:    cfg.Screenshot,
		Voice:         cfg.Voice,
		McpServerRoot: cfg.McpServerRoot,
		McpServers:    cfg.McpServers,
		Format:        cfg.Format,
		Output:        cfg.Output,
		Me:            cfg.Me,
		Template:      cfg.Template,
		New:           cfg.New,
		MaxHistory:    cfg.MaxHistory,
		MaxSpan:       cfg.MaxSpan,
		History:       cfg.History,
		Models:        cfg.Models,
		Log:           cfg.Log,
		Debug:         cfg.Debug,
		Quiet:         cfg.Quiet,
		Internal:      cfg.Internal,
		DenyList:      append([]string(nil), cfg.DenyList...),
		AllowList:     append([]string(nil), cfg.AllowList...),
		Unsafe:        cfg.Unsafe,
		Base:          cfg.Base,
		Workspace:     cfg.Workspace,
		Home:          cfg.Home,
		Temp:          cfg.Temp,
		Interactive:   cfg.Interactive,
		Editing:       cfg.Editing,
		Shell:         cfg.Shell,
		Watch:         cfg.Watch,
		ClipWatch:     cfg.ClipWatch,
		Hub:           cfg.Hub,
		MaxTime:       cfg.MaxTime,
		MaxTurns:      cfg.MaxTurns,
		Stdout:        cfg.Stdout,
		Stderr:        cfg.Stderr,
	}
}

func LoadModels(base string) (map[string]*model.ModelsConfig, error) {
	m, err := model.LoadModels(base)
	if err != nil {
		return nil, err
	}
	return m, err
}

func LoadHistory(base string, maxHistory, maxSpan int) ([]*Message, error) {
	if maxHistory <= 0 || maxSpan <= 0 {
		return nil, nil
	}

	var history []*Message

	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	// Collect .json files and their infos
	type fileInfo struct {
		name string
		mod  time.Time
	}
	var files []fileInfo

	old := time.Now().Add(-time.Duration(maxSpan) * time.Minute)

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			fullPath := filepath.Join(base, entry.Name())
			info, err := os.Stat(fullPath)
			if err == nil {
				if info.ModTime().Before(old) {
					continue
				}
				files = append(files, fileInfo{name: fullPath, mod: info.ModTime()})
			}
		}
	}

	// Sort by mod time DESC (most recent first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].mod.After(files[j].mod)
	})

	for _, fi := range files {
		data, err := os.ReadFile(fi.name)
		if err != nil {
			continue
		}
		var msgs []*Message
		if err := json.Unmarshal(data, &msgs); err != nil {
			continue
		}
		for i := len(msgs) - 1; i >= 0; i-- {
			history = append(history, msgs[i])
			if maxHistory > 0 && len(history) >= maxHistory {
				result := history[:maxHistory]
				reverseMessages(result)
				return result, nil
			}
		}
	}

	reverseMessages(history)
	return history, nil
}

func reverseMessages(msgs []*Message) {
	for left, right := 0, len(msgs)-1; left < right; left, right = left+1, right-1 {
		msgs[left], msgs[right] = msgs[right], msgs[left]
	}
}

func (r *AppConfig) StoreHistory(messages []*Message) error {
	dir := filepath.Join(filepath.Dir(r.ConfigFile), "history")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// filename
	now := time.Now()
	filename := fmt.Sprintf("%s-%d.json", now.Format("2006-01-02"), now.UnixNano())
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
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

type GitConfig struct {
}

type DBCred struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
}

// DSN returns the data source name for connecting to the database.
func (d *DBCred) DSN() string {
	host := d.Host
	if host == "" {
		host = "localhost"
	}
	port := d.Port
	if port == "" {
		port = "5432"
	}
	dbname := d.DBName
	if dbname == "" {
		dbname = "postgres"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, d.Username, d.Password, dbname)
}

func (d *DBCred) IsValid() bool {
	return d.Username != "" && d.Password != ""
}

func (d *DBCred) Clone() *DBCred {
	return &DBCred{
		Host:     d.Host,
		Port:     d.Port,
		Username: d.Username,
		Password: d.Password,
		DBName:   d.DBName,
	}
}

type TTSConfig struct {
	Provider string

	Model   string
	BaseUrl string
	ApiKey  string
}

func (config *TTSConfig) Clone() *TTSConfig {
	return &TTSConfig{
		Provider: config.Provider,
		Model:    config.Model,
		BaseUrl:  config.BaseUrl,
		ApiKey:   config.ApiKey,
	}
}

type HubConfig struct {
	Enable  bool
	Address string

	Pg        bool
	PgAddress string

	Mysql        bool
	MysqlAddress string

	Redis        bool
	RedisAddress string

	Terminal        bool
	TerminalAddress string
}
