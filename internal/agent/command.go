package agent

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/cb"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func AgentHelp(cfg *internal.AppConfig) error {
	log.Debugln("Agent smart help")

	in, err := GetUserInput(cfg)
	if err != nil {
		return err
	}
	if in.IsEmpty() {
		return internal.NewUserInputError("no query provided")
	}

	agent, err := NewHelpAgent(cfg)
	if err != nil {
		return err
	}

	if internal.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", internal.DryRunContent)
	}
	log.Infof("Sending request to [%s] %s...\n", cfg.LLM.Model, cfg.LLM.BaseUrl)

	// // Let LLM decide which agent to use
	next := func(ctx context.Context, req *api.Request) (*api.Response, error) {
		err := handleAgent(cfg, req)
		return nil, err
	}

	_, err = agent.Handle(context.TODO(), in, next)
	if err != nil {
		return err
	}

	log.Debugf("Agent help completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func handleAgent(cfg *internal.AppConfig, in *UserInput) error {
	agent, err := MakeAgent(in.Agent, cfg)
	if err != nil {
		return err
	}

	if internal.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", internal.DryRunContent)
	}
	log.Infof("[%s/%s] sending request to [%s] %s...\n", in.Agent, in.Subcommand, cfg.LLM.Model, cfg.LLM.BaseUrl)

	ctx := context.TODO()
	resp, err := agent.Send(ctx, in)
	if err != nil {
		return err
	}
	processContent(cfg, resp)

	log.Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func Info(cfg *internal.AppConfig) error {
	info, err := collectSystemInfo()
	if err != nil {
		log.Errorln(err)
		return err
	}
	log.Infoln(info)
	return nil
}

func Setup(cfg *internal.AppConfig) error {
	if err := setupConfig(cfg); err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}
	return nil
}

func ListAgents(cfg *internal.AppConfig) error {
	const format = `
Available agents:

%s

/ is shorthand for @script

Not sure which agent to use? Simply enter your message and AI will choose the most appropriate one for you:

ai "message..."

`
	dict := agentList()
	var keys []string
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	commands := agentCommandList()
	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteString(":\t")
		buf.WriteString(dict[k])
		if v, ok := commands[k]; ok {
			buf.WriteString("\t")
			buf.WriteString(v)
		}
		buf.WriteString("\n")
	}
	log.Printf(format, buf.String())

	return nil
}

func ListCommands(cfg *internal.AppConfig) error {
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

func processContent(cfg *internal.AppConfig, message *ChatMessage) {
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

	PrintMessage(cfg.Format, message)
	if cfg.Output != "" {
		SaveMessage(cfg.Output, message)
	}

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
