package agent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/cb"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/tool"
	"github.com/qiangli/ai/internal/util"
)

func AgentHelp(cfg *llm.Config, role, content string) error {
	log.Debugln("Agent smart help")
	msg := strings.TrimSpace(strings.Join(cfg.Args, " "))
	if msg == "" {
		return internal.NewUserInputError("no query provided")
	}

	agent, err := NewHelpAgent(cfg, role, content)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
	}
	log.Infof("Sending request to [%s] %s...\n", cfg.Model, cfg.BaseUrl)

	ctx := context.TODO()
	resp, err := agent.Send(ctx, msg)
	if err != nil {
		return err
	}

	// clone the cfg to avoid modifying the original one
	nc := cfg.Clone()
	if err := dispatch(nc, resp, role, content); err != nil {
		return err
	}

	log.Debugf("Agent help completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func AgentCommand(cfg *llm.Config, role, content string) error {
	log.Debugf("Agent: %s %v\n", cfg.Command, cfg.Args)

	name := strings.TrimSpace(cfg.Command[1:])
	if name == "" {
		return internal.NewUserInputError("no agent provided")
	}

	dict, err := ListAgents()
	if err != nil {
		return err
	}
	if _, exist := dict[name]; !exist {
		return internal.NewUserInputError("no such agent: " + name)
	}

	msg, err := GetUserInput(cfg)
	if err != nil {
		return err
	}
	if msg == "" {
		return internal.NewUserInputError("no message content")
	}

	agent, err := MakeAgent(name, cfg, role, content)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
	}
	log.Infof("Sending request to [%s] %s...\n", cfg.Model, cfg.BaseUrl)

	ctx := context.TODO()
	resp, err := agent.Send(ctx, msg)
	if err != nil {
		return err
	}
	processContent(cfg, resp)

	log.Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func SlashCommand(cfg *llm.Config, role, content string) error {
	log.Debugf("Command: %s %v\n", cfg.Command, cfg.Args)

	name := strings.TrimSpace(cfg.Command[1:])
	if name != "" {
		name = filepath.Base(name)
	}

	msg, err := GetUserInput(cfg)
	if err != nil {
		return err
	}

	if name == "" && msg == "" {
		return internal.NewUserInputError("no command and message provided")
	}

	agent, err := NewScriptAgent(cfg, role, content)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
	}
	log.Infof("Sending request to [%s] %s...\n", cfg.Model, cfg.BaseUrl)

	ctx := context.TODO()
	resp, err := agent.Send(ctx, name, msg)
	if err != nil {
		return err
	}
	processContent(cfg, resp)

	log.Debugf("Command completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func InfoCommand(cfg *llm.Config) error {
	info, err := collectSystemInfo()
	if err != nil {
		log.Errorln(err)
		return err
	}
	log.Infoln(info)
	return nil
}

func ListCommand(cfg *llm.Config) error {
	list, err := tool.ListCommands(false)
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

	// show content to the stdout
	showContent := func() {
		log.Infof("\n[%s]\n", message.Agent)
		log.Println(content)
	}

	// process code blocks
	if total > 0 {
		if cfg.Interactive {
			showContent()

			log.Infof("\n=== CODE BLOCKS (%v) ===\n", total)
			for i, v := range doc.CodeBlocks {
				log.Infof("\n===\n%s\n=== %v/%v ===\n", v.Code, i+1, total)
				ProcessBashScript(cfg, v.Code)
			}
			log.Infoln("=== END ===\n")
		} else {
			// if there are code blocks in non-interactive mode
			// we don't show the content to stdout
			// this is to ensure the code blocks can be piped/redirected
			// without being mixed with other content
			log.Infof("\n[%s]\n", message.Agent)
			log.Infoln(content)

			const codeTpl = "%s\n"
			var snippets []string
			for _, v := range doc.CodeBlocks {
				snippets = append(snippets, v.Code)
			}
			// show code snippets
			log.Printf(codeTpl, strings.Join(snippets, "\n"))
		}
	} else {
		showContent()
	}
}

func dispatch(cfg *llm.Config, resp *ChatMessage, role, content string) error {
	log.Debugf("dispatching: %+v\n", resp)

	var data map[string]string
	if err := json.Unmarshal([]byte(resp.Content), &data); err != nil {
		return err
	}
	what := data["type"]
	arg := data["arg"]

	switch what {
	case "command":
		if arg == "/" {
			cfg.Command = "/"
		} else {
			cfg.Command = "/" + arg
		}
		log.Infof("Running: ai %s...\n", cfg.Command)
		return SlashCommand(cfg, role, content)
	case "agent":
		cfg.Command = "@" + arg
		log.Infof("Running: ai %s...\n", cfg.Command)
		return AgentCommand(cfg, role, content)
	default:
		log.Debugf("unknown type: %s, default to '@ask'\n", what)
		cfg.Command = "@ask"
		log.Infof("Running: ai %s...\n", cfg.Command)
		return AgentCommand(cfg, role, content)
	}
}
