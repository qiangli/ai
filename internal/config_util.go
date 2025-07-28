package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	fangs "github.com/spf13/viper"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/api/model"
)

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
	var lc = &api.LLMConfig{}
	app.LLM = lc
	// default
	lc.Provider = viper.GetString("provider")

	lc.ApiKey = viper.GetString("api_key")
	lc.Model = viper.GetString("model")
	lc.BaseUrl = viper.GetString("base_url")

	// <provider>/<model>
	modelName := func(n string) string {
		if strings.Contains(n, "/") {
			return n
		}
		if lc.Provider == "" {
			return "openai/" + n
		}
		return lc.Provider + "/" + n
	}

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
	app.Models = alias

	// load models from app base
	modelBase := filepath.Join(app.Base, "models")
	modelCfg, err := model.LoadModels(modelBase)
	if err != nil {
		return err
	}
	if alias != "" {
		if m, ok := modelCfg[alias]; ok {
			app.LLM.Models = m.Models
		}
	}

	// if no models, setup defaults
	if len(app.LLM.Models) == 0 {
		// all levels share same config
		var m model.Model
		switch {
		case lc.ApiKey != "" && lc.Model != "":
			// assume openai compatible
			m = model.Model{
				Name:    modelName(lc.Model),
				BaseUrl: lc.BaseUrl,
				ApiKey:  lc.ApiKey,
			}
		case os.Getenv("OPENAI_API_KEY") != "":
			m = model.Model{
				Name:    "openai/gpt-4.1-mini",
				BaseUrl: "https://api.openai.com/v1/",
				ApiKey:  os.Getenv("OPENAI_API_KEY"),
			}
		case os.Getenv("GEMINI_API_KEY") != "":
			m = model.Model{
				Name:    "gemini/gemini-2.0-flash-lite",
				BaseUrl: "",
				ApiKey:  os.Getenv("GEMINI_API_KEY"),
			}
		case os.Getenv("ANTHROPIC_API_KEY") != "":
			m = model.Model{
				Name:    "anthropic/claude-3-5-haiku-latest",
				BaseUrl: "",
				ApiKey:  os.Getenv("ANTHROPIC_API_KEY"),
			}
		default:
		}

		// TODO improve to allow any alias other than L*
		models := make(map[model.Level]*model.Model)
		models[model.L1] = m.Clone()
		models[model.L2] = m.Clone()
		models[model.L3] = m.Clone()

		app.LLM.Models = models
	}
	// update or add model from command line flags
	for _, l := range model.Levels {
		s := strings.ToLower(string(l))
		k := viper.GetString(s + "_api_key")
		n := viper.GetString(s + "_model")
		u := viper.GetString(s + "_base_url")
		if v, ok := app.LLM.Models[l]; ok {
			if k != "" {
				v.ApiKey = k
			}
			if n != "" {
				v.Name = modelName(n)
			}
			if u != "" {
				v.BaseUrl = u
			}
			app.LLM.Models[l] = v
		} else {
			app.LLM.Models[l] = &model.Model{
				Name:    modelName(n),
				ApiKey:  k,
				BaseUrl: u,
			}
		}
	}
	// model config is required
	if len(app.LLM.Models) == 0 {
		return fmt.Errorf("No LLM configuration found")
	}

	// TODO
	tts := &api.TTSConfig{}
	tts.ApiKey = viper.GetString("tts_api_key")
	tts.Provider = viper.GetString("tts_provider")
	tts.Model = viper.GetString("tts_model")
	tts.BaseUrl = viper.GetString("tts_base_url")
	if tts.ApiKey == "" {
		tts.ApiKey = os.Getenv("OPENAI_API_KEY")
	}
	if tts.Model == "" {
		tts.Model = "gpt-4o-mini-tts"
	}

	app.TTS = tts

	return nil
}
