package internal

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	fangs "github.com/spf13/viper"
	"github.com/tailscale/hujson"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

const DefaultEditor = "ai -i edit"

const StdinRedirect = "-"

// clipboard redirection
const (
	// read from clipboard
	ClipinRedirect = "{"

	// read from clipboard and wait, allowing multiple copy
	// until Ctrl-D is entered
	ClipinRedirect2 = "{{"

	// write to clipboard, overwriting its content
	ClipoutRedirect = "}"

	// append to clipboard
	ClipoutRedirect2 = "}}"
)

var Version = "0.0.1" // version of the ai binary

//go:embed data/*
var configData embed.FS

func GetConfigData() embed.FS {
	return configData
}

// global flags

var V *fangs.Viper

func init() {
	V = fangs.New()
}

// init viper
func InitConfig(viper *fangs.Viper) {
	defaultCfg := os.Getenv("AI_CONFIG")
	if defaultCfg == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultCfg = filepath.Join(home, ".ai", "config.yaml")
		}
	}
	if defaultCfg != "" {
		viper.SetConfigFile(defaultCfg)
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Debugf("Error reading config file: %s\n", err)
	}
}

func ParseConfig(viper *fangs.Viper, app *api.AppConfig, args []string) error {
	app.ConfigFile = viper.ConfigFileUsed()
	app.Base = filepath.Dir(app.ConfigFile)

	log.Debugf("configure file: %s\n", app.ConfigFile)
	log.Debugf("base: %s\n", app.Base)
	printAIEnv()

	//

	app.Version = Version

	// app.Role = viper.GetString("role")
	// app.Prompt = viper.GetString("prompt")

	app.DryRun = viper.GetBool("dry_run")
	app.DryRunContent = viper.GetString("dry_run_content")

	// TODO read from user.json
	user := getCurrentUser()
	app.Me = &api.User{
		Username: user,
		Display:  "👤 " + strings.ToUpper(user),
	}

	app.Files = viper.GetStringSlice("file")
	app.Format = viper.GetString("format")
	app.Output = viper.GetString("output")

	//
	app.Message = viper.GetString("message")

	app.Template = viper.GetString("template")

	app.Screenshot = viper.GetBool("screenshot")
	app.Voice = viper.GetBool("voice")

	//
	home, err := homeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	app.Home = home

	temp, err := tempDir()
	if err != nil {
		return fmt.Errorf("failed to get temp directory: %w", err)
	}
	app.Temp = temp

	ws := viper.GetString("workspace")
	ws, err = resolveWorkspaceDir(ws)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}
	app.Workspace = ws

	//
	app.New = viper.GetBool("new")
	app.ChatID = viper.GetString("chat")
	app.MaxHistory = viper.GetInt("max_history")
	app.MaxSpan = viper.GetInt("max_span")

	app.MaxTurns = viper.GetInt("max_turns")
	app.MaxTime = viper.GetInt("max_time")

	//
	if !app.New {
		baseDir := filepath.Join(app.Base, "chat")

		var cid = app.ChatID
		if cid == "" {
			if last, err := api.FindLastChatID(baseDir); err == nil {
				cid = last
			}
		}

		if cid != "" {
			chatDir := filepath.Join(baseDir, cid)
			log.Debugf("Loading old conversation from %s", chatDir)
			messages, err := api.LoadHistory(chatDir, app.MaxHistory, app.MaxSpan)
			if err != nil {
				return fmt.Errorf("error loading history: %v", err)
			}
			app.ChatID = cid
			app.History = messages
		}
	}

	// ensure chat id is assigned
	if app.ChatID == "" {
		app.ChatID = uuid.New().String()
	}

	if err := ParseLLM(viper, app); err != nil {
		return err
	}

	//
	app.Log = viper.GetString("log")
	app.Trace = viper.GetBool("trace")
	app.Debug = viper.GetBool("verbose")
	app.Quiet = viper.GetBool("quiet")
	app.Internal = viper.GetBool("internal")

	app.Unsafe = viper.GetBool("unsafe")
	toList := func(s string) []string {
		sa := strings.Split(s, ",")
		var list []string
		for _, v := range sa {
			list = append(list, strings.TrimSpace(v))
		}
		if len(list) > 0 {
			return list
		}
		return nil
	}
	app.DenyList = toList(viper.GetString("deny"))
	app.AllowList = toList(viper.GetString("allow"))

	app.Editor = viper.GetString("editor")
	app.Editing = viper.GetBool("edit")
	app.Interactive = viper.GetBool("interactive")

	app.Watch = viper.GetBool("watch")
	app.ClipWatch = viper.GetBool("pb_watch")

	shell := viper.GetString("shell")
	if shell == "" {
		shell = "bash"
	}
	shellBin, _ := exec.LookPath(shell)
	if shellBin == "" {
		shellBin = "/bin/bash"
	}
	app.Shell = shellBin

	// default agent:
	// --agent, "ask"
	var defaultAgent = viper.GetString("agent")
	if defaultAgent == "" {
		defaultAgent = "ask"
	}

	//
	ParseArgs(viper, app, args, defaultAgent)

	// mcp
	app.McpServerRoot = viper.GetString("mcp.server_root")
	if app.McpServerRoot != "" {
		mcp := NewMcpServersConfig(app.McpServerRoot)
		if err := mcp.LoadAll(); err != nil {
			return err
		}
		app.McpServers = mcp.ServersConfig
	}

	// resource
	resource := viper.GetString("resource")
	if resource != "" {
		ar, err := api.LoadAgentResource(filepath.Join(app.Base, resource))
		if err != nil {
			return err
		}
		app.AgentResource = ar
	}

	// sql db
	// dbCfg := &api.DBCred{}
	// dbCfg.Host = viper.GetString("sql.db_host")
	// dbCfg.Port = viper.GetString("sql.db_port")
	// dbCfg.Username = viper.GetString("sql.db_username")
	// dbCfg.Password = viper.GetString("sql.db_password")
	// dbCfg.DBName = viper.GetString("sql.db_name")
	// app.DBCred = dbCfg
	//
	// gitConfig := &api.GitConfig{}
	// app.Git = gitConfig

	log.Debug("parsed: %+v\n", app)

	return nil
}

func ParseArgs(viper *fangs.Viper, app *api.AppConfig, args []string, defaultAgent string) {
	newArgs := ParseAgentArgs(app, args, defaultAgent)
	newArgs = ParseSpecialChars(viper, app, newArgs)
	app.Args = newArgs
}

type McpServersConfig struct {
	ServersRoot string

	ServersConfig map[string]*api.McpServerConfig `json:"mcpServers"`
}

func NewMcpServersConfig(root string) *McpServersConfig {
	return &McpServersConfig{
		ServersRoot:   root,
		ServersConfig: make(map[string]*api.McpServerConfig),
	}
}

func (r *McpServersConfig) LoadAll() error {
	entries, err := os.ReadDir(r.ServersRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			name := entry.Name()
			ext := filepath.Ext(name)
			if ext == ".jsonc" || ext == ".json" {
				configPath := filepath.Join(r.ServersRoot, name)
				if err := r.LoadFile(configPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *McpServersConfig) LoadFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return r.LoadData(data)
}

func (r *McpServersConfig) LoadData(data []byte) error {
	hu, err := hujson.Standardize(data)
	if err != nil {
		return err
	}
	ex := expandWithDefault(string(hu))
	err = json.Unmarshal([]byte(ex), r)
	if err != nil {
		return fmt.Errorf("unmarshal mcp config: %v", err)
	}

	// set server name for each config
	for k, v := range r.ServersConfig {
		v.Server = k
	}
	return nil
}

// return the agent/command and the rest of the args
func ParseAgentArgs(app *api.AppConfig, args []string, defaultAgent string) []string {
	shellAgent := "shell"

	// first or last arg could be the agent/command
	// the last takes precedence
	var arg string
	isAgent := func(s string) bool {
		return strings.HasPrefix(s, "@")
	}
	isSlash := func(s string) bool {
		return strings.HasPrefix(s, "/")
	}
	switch len(args) {
	case 0:
		// no args, use default agent
	case 1:
		if isSlash(args[0]) || isAgent(args[0]) {
			arg = args[0]
			args = args[1:]
		}
	default:
		if isSlash(args[0]) || isAgent(args[0]) {
			arg = args[0]
			args = args[1:]
		}
		// agent check only
		// slash could file path
		if isAgent(args[len(args)-1]) {
			arg = args[len(args)-1]
			args = args[:len(args)-1]
		}
	}

	var agent string
	if arg != "" {
		if arg[0] == '/' {
			agent = shellAgent + arg
		} else {
			agent = arg[1:]
		}
	}

	if agent == "" {
		agent = defaultAgent
	}

	parts := strings.SplitN(agent, "/", 2)
	app.Agent = parts[0]
	if len(parts) > 1 {
		app.Command = parts[1]
	}

	return args
}

// parse special char sequence for stdin/clipboard
// they can:
// + be at the end of the args or as a suffix to the last one
// + be in any order
// + be multiple instances
func ParseSpecialChars(viper *fangs.Viper, app *api.AppConfig, args []string) []string {
	// special char sequence handling
	var stdin = viper.GetBool("stdin")
	var pbRead = viper.GetBool("pb_read")
	var pbReadWait = viper.GetBool("pb_tail")
	var pbWrite = viper.GetBool("pb_write")
	var pbWriteAppend = viper.GetBool("pb_append")
	var isStdin, isClipin, isClipWait, isClipout, isClipAppend bool

	newArgs := make([]string, len(args))

	if len(args) > 0 {
		for i := len(args) - 1; i >= 0; i-- {
			lastArg := args[i]

			if lastArg == StdinRedirect {
				isStdin = true
			} else if lastArg == ClipinRedirect {
				isClipin = true
			} else if lastArg == ClipinRedirect2 {
				isClipin = true
				isClipWait = true
			} else if lastArg == ClipoutRedirect {
				isClipout = true
			} else if lastArg == ClipoutRedirect2 {
				isClipout = true
				isClipAppend = true
			} else {
				// check for suffix for cases where the special char is not the last arg
				// but is part of the last arg
				if strings.HasSuffix(lastArg, StdinRedirect) {
					isStdin = true
					args[i] = strings.TrimSuffix(lastArg, StdinRedirect)
				} else if strings.HasSuffix(lastArg, ClipinRedirect) {
					isClipin = true
					args[i] = strings.TrimSuffix(lastArg, ClipinRedirect)
				} else if strings.HasSuffix(lastArg, ClipinRedirect2) {
					isClipin = true
					isClipWait = true
					args[i] = strings.TrimSuffix(lastArg, ClipinRedirect2)
				} else if strings.HasSuffix(lastArg, ClipoutRedirect) {
					isClipout = true
					args[i] = strings.TrimSuffix(lastArg, ClipoutRedirect)
				} else if strings.HasSuffix(lastArg, ClipoutRedirect2) {
					isClipout = true
					isClipAppend = true
					args[i] = strings.TrimSuffix(lastArg, ClipoutRedirect2)
				}
				newArgs = args[:i+1]
				break
			}
		}
	}

	isPiped := func() bool {
		stat, _ := os.Stdin.Stat()
		return (stat.Mode() & os.ModeCharDevice) == 0
	}

	app.IsPiped = isPiped()
	app.Stdin = isStdin || stdin
	app.Clipin = isClipin || pbRead || pbReadWait
	app.ClipWait = isClipWait || pbReadWait
	app.Clipout = isClipout || pbWrite || pbWriteAppend
	app.ClipAppend = isClipAppend || pbWriteAppend

	return newArgs
}

func ParseLLM(viper *fangs.Viper, app *api.AppConfig) error {
	// LLM config
	// var lc = &api.LLMConfig{}
	// app.LLM = lc
	// default
	// lc.Provider = viper.GetString("provider")

	// lc.ApiKey = viper.GetString("api_key")
	// lc.Name = viper.GetString("model")
	// lc.BaseUrl = viper.GetString("base_url")

	// // <provider>/<model>
	// modelName := func(n string) string {
	// 	if strings.Contains(n, "/") {
	// 		return n
	// 	}
	// 	if lc.Provider == "" {
	// 		return "openai/" + n
	// 	}
	// 	return lc.Provider + "/" + n
	// }

	//
	alias := viper.GetString("models")
	// use same models to continue the conversation
	// if not set
	if alias == "" {
		if len(app.History) > 0 {
			last := app.History[len(app.History)-1]
			alias = last.Models
		}
	}
	if alias == "" {
		alias = "openai"
	}
	app.Models = alias

	return nil
}
