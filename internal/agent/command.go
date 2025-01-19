package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/cb"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func AgentHelp(cfg *llm.Config, role, prompt string) error {
	log.Debugln("Agent smart help")

	in, err := GetUserInput(cfg)
	if err != nil {
		return err
	}
	if in.IsEmpty() {
		return internal.NewUserInputError("no query provided")
	}

	agent, err := NewHelpAgent(cfg, role, prompt)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
	}
	log.Infof("Sending request to [%s] %s...\n", cfg.Model, cfg.BaseUrl)

	// Let LLM decide which agent to use
	ctx := context.TODO()
	resp, err := agent.Send(ctx, in.Input())
	if err != nil {
		return err
	}

	// clone the cfg to avoid modifying the original one
	nc := cfg.Clone()
	if err := dispatch(nc, resp, role, prompt, in); err != nil {
		return err
	}

	log.Debugf("Agent help completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func AgentCommand(cfg *llm.Config, role, prompt string) error {
	log.Debugf("Agent: %s %v\n", cfg.Command, cfg.Args)

	names := strings.TrimSpace(cfg.Command[1:])
	if names == "" {
		return internal.NewUserInputError("no agent provided")
	}

	na := strings.SplitN(names, "/", 2)
	name := na[0]

	if !hasAgent(name) {
		return internal.NewUserInputError("no such agent: " + name)
	}
	input, err := GetUserInput(cfg)
	if err != nil {
		return err
	}
	if input.IsEmpty() {
		return internal.NewUserInputError("no message content")
	}

	input.Command = name
	if len(na) > 1 {
		input.SubCommand = na[1]
	}

	return handleAgent(cfg, role, prompt, input)
}

func handleAgent(cfg *llm.Config, role, prompt string, input *UserInput) error {

	agent, err := MakeAgent(input.Command, cfg, role, prompt)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
	}
	log.Infof("Sending request to [%s] %s...\n", cfg.Model, cfg.BaseUrl)

	ctx := context.TODO()
	resp, err := agent.Send(ctx, input)
	if err != nil {
		return err
	}
	processContent(cfg, resp)

	log.Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func SlashCommand(cfg *llm.Config, role, prompt string) error {
	log.Debugf("Command: %s %v\n", cfg.Command, cfg.Args)

	name := strings.TrimSpace(cfg.Command[1:])
	if name != "" {
		name = filepath.Base(name)
	}

	input, err := GetUserInput(cfg)
	if err != nil {
		return err
	}

	if name == "" && input.IsEmpty() {
		return internal.NewUserInputError("no command and message provided")
	}

	input.Command = name
	return handleSlash(cfg, role, prompt, input)
}

func handleSlash(cfg *llm.Config, role, prompt string, in *UserInput) error {
	agent, err := NewScriptAgent(cfg, role, prompt)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
	}
	log.Infof("Sending request to [%s] %s...\n", cfg.Model, cfg.BaseUrl)

	ctx := context.TODO()
	resp, err := agent.Send(ctx, in)
	if err != nil {
		return err
	}
	processContent(cfg, resp)

	log.Debugf("Command completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func Info(cfg *llm.Config) error {
	info, err := collectSystemInfo()
	if err != nil {
		log.Errorln(err)
		return err
	}
	log.Infoln(info)
	return nil
}

func Setup(cfg *llm.Config) error {
	if err := setupConfig(cfg); err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}
	return nil
}

func ListAgents(cfg *llm.Config) error {
	dict, err := agentList()
	if err != nil {
		return err
	}

	var keys []string
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		log.Printf("%s:\t%s\n", k, dict[k])
	}
	return nil
}

func ListCommands(cfg *llm.Config) error {
	list, err := util.ListCommands(false)
	if err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}

	const listTpl = `Available commands on the system:
%s
Total: %v
`
	sort.Strings(list)
	log.Printf(listTpl, strings.Join(list, "\n"), len(list))
	return nil
}

func collectSystemInfo() (string, error) {
	info, err := util.CollectSystemInfo()
	if err != nil {
		return "", err
	}
	jd, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jd), nil
}

func processContent(cfg *llm.Config, message *ChatMessage) {
	content := message.Content
	doc := util.ParseMarkdown(content)
	total := len(doc.CodeBlocks)

	// clipboard
	if cfg.Clipout {
		if err := cb.NewClipboard().Write(content); err != nil {
			log.Debugf("failed to copy content to clipboard: %v\n", err)
		}
	}

	// process code blocks
	isPiped := func() bool {
		stat, err := os.Stdout.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) == 0
	}()

	// TODO: markdown formatting lost if the content is also tee'd to a file
	renderMessage(message)

	if total > 0 && isPiped {
		// if there are code blocks and stdout is redirected
		// we send the code blocks to the stdout
		const codeTpl = "%s\n"
		var snippets []string
		for _, v := range doc.CodeBlocks {
			snippets = append(snippets, v.Code)
		}
		// show code snippets
		log.Printf(codeTpl, strings.Join(snippets, "\n"))
	}
}

func dispatch(cfg *llm.Config, resp *ChatMessage, role, prompt string, in *UserInput) error {
	log.Debugf("dispatching: %+v\n", resp)

	var data map[string]string
	if err := json.Unmarshal([]byte(resp.Content), &data); err != nil {
		// better continue the conversation than err
		log.Debugf("failed to unmarshal content: %v\n", err)
		data = map[string]string{"type": "agent", "arg": "ask"}
	}
	what := data["type"]
	name := data["arg"]

	switch what {
	case "command":
		if name == "/" {
			cfg.Command = "/"
		} else {
			cfg.Command = "/" + name
		}
		log.Infof("Running `ai %s` ...\n", cfg.Command)
		in.Command = name
		return handleSlash(cfg, role, prompt, in)
	case "agent":
		cfg.Command = "@" + name
		log.Infof("Running `ai %s` ...\n", cfg.Command)
		in.Command = name
		return handleAgent(cfg, role, prompt, in)
	default:
		log.Debugf("unknown type: %s, default to '@ask'\n", what)
		cfg.Command = "@ask"
		log.Infof("Running `ai %s` ...\n", cfg.Command)
		in.Command = "ask"
		return handleAgent(cfg, role, prompt, in)
	}
}
