package internal

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/tool"
	"github.com/qiangli/ai/internal/util"
)

// DetectContentType determines the content type of a file based on magic numbers, content, and file extension.
func DetectContentType(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("error opening file: %v", err)
		return ""
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		log.Errorf("error reading file: %v", err)
		return ""
	}

	contentType := http.DetectContentType(buffer)
	if contentType != "application/octet-stream" {
		return contentType
	}

	ext := filepath.Ext(filePath)
	if ext != "" {
		extType := mime.TypeByExtension(ext)
		if extType != "" {
			return extType
		}
	}

	return "application/octet-stream"
}

func AgentCommand(cfg *Config, role, content string) error {
	log.Debugf("Agent command: %s %v\n", cfg.Command, cfg.Args)

	name := strings.TrimSpace(cfg.Command[1:])
	if name == "" {
		return NewUserInputError("no agent provided")
	}

	dict, err := ListAgents()
	if err != nil {
		return err
	}
	if _, exist := dict[name]; !exist {
		return NewUserInputError("no such agent: " + name)
	}

	msg, err := GetUserInput(cfg)
	if err != nil {
		return err
	}
	if msg == "" {
		return NewUserInputError("no message content")
	}

	switch name {
	case "ask":
		agent, err := NewChat(cfg, role, content)
		if err != nil {
			return err
		}

		if cfg.DryRun {
			log.Infof("Dry run mode. No API call will be made!\n")
			log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
		}
		log.Infof("Sending request to %s...\n", cfg.BaseUrl)

		ctx := context.TODO()
		resp, err := agent.Send(ctx, msg)
		if err != nil {
			return err
		}
		processContent(cfg, resp.Content)
	default:
	}

	log.Debugf("agent command completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func SlashCommand(cfg *Config, role, content string) error {
	log.Debugf("Slash command: %s %v\n", cfg.Command, cfg.Args)

	name := strings.TrimSpace(cfg.Command[1:])
	if name != "" {
		name = filepath.Base(name)
	}

	msg, err := GetUserInput(cfg)
	if err != nil {
		return err
	}

	if name == "" && msg == "" {
		return NewUserInputError("no command and message provided")
	}

	agent, err := NewScriptAgent(cfg, role, content)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		log.Infof("Dry run mode. No API call will be made!\n")
		log.Debugf("The following will be returned:\n%s\n", cfg.DryRunContent)
	}
	log.Infof("Sending request to %s...\n", cfg.BaseUrl)

	ctx := context.TODO()
	resp, err := agent.Send(ctx, name, msg)
	if err != nil {
		return err
	}
	processContent(cfg, resp.Content)

	log.Debugf("Slash command completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func InfoCommand(cfg *Config) error {
	info, err := collectSystemInfo()
	if err != nil {
		log.Errorln(err)
		return err
	}
	log.Infoln(info)
	return nil
}

func ListCommand(cfg *Config) error {
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

func processContent(cfg *Config, content string) {
	doc := ParseMarkdown(content)
	total := len(doc.CodeBlocks)

	// clipboard
	if cfg.Clipout {
		if err := NewClipboard().Write(content); err != nil {
			log.Debugf("failed to copy content to clipboard: %v\n", err)
		}
	}

	// show content to the stdout
	showContent := func() {
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
