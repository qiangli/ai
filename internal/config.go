package internal

import (
	_ "embed"
	"errors"
	"os"
	"os/user"
	"strings"

	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal/api"
)

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

// return the agent/command and the rest of the args
func parseArgs(app *AppConfig, args []string) []string {
	agent := viper.GetString("agent")
	if agent == "" {
		agent = "ask"
	}
	newArgs := make([]string, 0, len(args))
	for i, arg := range args {
		if arg[0] == '/' || arg[0] == '@' {
			if arg[0] == '/' {
				agent = "script" + arg
			} else {
				agent = arg[1:]
			}
			newArgs = append(newArgs, args[i+1:]...)
			break
		}
		newArgs = append(newArgs, arg)
	}
	tool := strings.SplitN(agent, "/", 2)

	app.Agent = tool[0]
	if len(tool) > 1 {
		app.Command = tool[1]
	}

	return newArgs
}

// parse special char sequence for stdin/clipboard: - = =+
// they can be:
//
//	at the end of the args or as a suffix to the last arg
//	in any order
//	multiple instances
func parseSpecialChars(app *AppConfig, args []string) []string {
	// special char sequence handling
	var stdin = viper.GetBool("stdin")
	var pbRead = viper.GetBool("pb_read")
	var pbReadWait = viper.GetBool("pb_read_wait")
	var pbWrite = viper.GetBool("pb_write")
	var pbWriteAppend = viper.GetBool("pb_write_append")
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

var ConfigFile string

var InputFiles []string
var FormatFlag string
var OutputFlag string
var TemplateFile string

type DBConfig = api.DBCred
type LLMConfig = api.LLMConfig

type GitConfig struct {
}

//go:embed ai.yaml
var configFileContent string

func GetDefaultConfig() string {
	return configFileContent
}

// global flags

var DryRun bool
var DryRunContent string

type AppConfig struct {
	Version string

	ConfigFile string

	LLM *LLMConfig

	Git *GitConfig
	Db  *DBConfig

	Role   string
	Prompt string

	Agent   string
	Command string
	Args    []string

	// --message takes precedence over all other forms of input
	Message string

	Editor string

	Clipin   bool
	ClipWait bool

	Clipout    bool
	ClipAppend bool

	IsPiped bool
	Stdin   bool

	Files []string

	// MCP server url
	McpServerUrl string

	// Output format: raw or markdown
	Format string

	// Save output to file
	Output string

	Me string

	//
	Template string

	Debug    bool
	Internal bool

	//
	Workspace string

	Interactive bool
	MetaPrompt  bool

	MaxTime  int
	MaxTurns int
}

func getCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return "ME"
	}
	return strings.ToUpper(currentUser.Username)
}

func ParseConfig(args []string) (*AppConfig, error) {
	var lc = &LLMConfig{}
	var app = &AppConfig{}

	app.LLM = lc
	app.Version = Version
	app.Role = viper.GetString("role")
	app.Prompt = viper.GetString("role_prompt")

	app.Me = "👤 " + getCurrentUser()
	app.Files = InputFiles
	app.Format = FormatFlag
	app.Output = OutputFlag
	app.ConfigFile = viper.ConfigFileUsed()

	app.Message = viper.GetString("message")
	// read input file if message is empty
	inputFile := viper.GetString("input")
	if inputFile != "" && app.Message == "" {
		b, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, errors.New("failed to read input file")
		} else {
			app.Message = string(b)
		}
	}
	app.Template = TemplateFile
	app.Workspace = viper.GetString("workspace")
	app.McpServerUrl = viper.GetString("mcp_server_url")

	// LLM config
	lc.ApiKey = viper.GetString("api_key")
	lc.Model = viper.GetString("model")
	lc.BaseUrl = viper.GetString("base_url")

	lc.L1Model = viper.GetString("l1_model")
	lc.L1BaseUrl = viper.GetString("l1_base_url")
	lc.L1ApiKey = viper.GetString("l1_api_key")
	lc.L2Model = viper.GetString("l2_model")
	lc.L2BaseUrl = viper.GetString("l2_base_url")
	lc.L2ApiKey = viper.GetString("l2_api_key")
	lc.L3Model = viper.GetString("l3_model")
	lc.L3BaseUrl = viper.GetString("l3_base_url")
	lc.L3ApiKey = viper.GetString("l3_api_key")
	if lc.L1Model == "" {
		lc.L1Model = lc.Model
	}
	if lc.L2Model == "" {
		lc.L2Model = lc.Model
	}
	if lc.L3Model == "" {
		lc.L3Model = lc.Model
	}
	if lc.L1ApiKey == "" {
		lc.L1ApiKey = lc.ApiKey
	}
	if lc.L2ApiKey == "" {
		lc.L2ApiKey = lc.ApiKey
	}
	if lc.L3ApiKey == "" {
		lc.L3ApiKey = lc.ApiKey
	}
	if lc.L1BaseUrl == "" {
		lc.L1BaseUrl = lc.BaseUrl
	}
	if lc.L2BaseUrl == "" {
		lc.L2BaseUrl = lc.BaseUrl
	}
	if lc.L3BaseUrl == "" {
		lc.L3BaseUrl = lc.BaseUrl
	}
	lc.ImageModel = viper.GetString("image_model")
	lc.ImageBaseUrl = viper.GetString("image_base_url")
	lc.ImageApiKey = viper.GetString("image_api_key")

	//
	app.Debug = viper.GetBool("verbose")
	app.Internal = viper.GetBool("internal")

	app.Editor = viper.GetString("editor")
	app.Interactive = viper.GetBool("interactive")
	noMeta := viper.GetBool("no_meta_prompt")
	app.MetaPrompt = !noMeta

	app.MaxTurns = viper.GetInt("max_turns")
	app.MaxTime = viper.GetInt("max_time")

	newArgs := parseArgs(app, args)

	newArgs = parseSpecialChars(app, newArgs)
	app.Args = newArgs

	// sql db
	dbCfg := &DBConfig{}
	dbCfg.Host = viper.GetString("sql.db_host")
	dbCfg.Port = viper.GetString("sql.db_port")
	dbCfg.Username = viper.GetString("sql.db_username")
	dbCfg.Password = viper.GetString("sql.db_password")
	dbCfg.DBName = viper.GetString("sql.db_name")
	app.Db = dbCfg
	//
	gitConfig := &GitConfig{}
	app.Git = gitConfig

	return app, nil
}

func (r *AppConfig) IsStdin() bool {
	return r.Stdin || r.IsPiped
}

func (r *AppConfig) IsSpecial() bool {
	return r.IsStdin() || r.Clipin
}

func (r *AppConfig) HasInput() bool {
	return r.Message != "" || len(r.Files) > 0 || len(r.Args) > 0
}
