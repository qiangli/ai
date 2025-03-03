package agent

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/cb"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
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

	in.Agent = launchAgent

	return RunSwarm(cfg, in)
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
	// dict := agentList()
	dict := resource.AgentCommandMap
	var keys []string
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// commands := agentCommandList()
	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteString(":\t")
		buf.WriteString(dict[k].Description)
		// if v, ok := commands[k]; ok {
		// 	buf.WriteString("\t")
		// 	buf.WriteString(v)
		// }
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

func ListTools(cfg *internal.AppConfig) error {
	list := []string{}
	const listTpl = `Available MCP tools:

%s

Total: %v
`
	// TODO flag to set the proxy url
	if os.Getenv("AI_MCP_PROXY_URL") == "" {
		os.Setenv("AI_MCP_PROXY_URL", "http://localhost:58080/sse")
	}

	tools, err := swarm.ListTools()
	if err != nil {
		return err
	}
	for k, tool := range tools {
		for _, v := range tool {
			list = append(list, fmt.Sprintf("%s: %s\n", k, strings.TrimSpace(v.Description)))
		}
	}

	sort.Strings(list)
	log.Printf(listTpl, strings.Join(list, "\n"), len(list))
	return nil
}

func processContent(cfg *internal.AppConfig, message *api.Response) {
	if message.ContentType == api.ContentTypeText || message.ContentType == "" {
		processTextContent(cfg, message)
	} else if message.ContentType == api.ContentTypeB64JSON {
		processImageContent(cfg, message)
	} else {
		log.Debugf("Unsupported content type: %s\n", message.ContentType)
	}
}

func processTextContent(cfg *internal.AppConfig, message *api.Response) {
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

func processImageContent(cfg *internal.AppConfig, message *api.Response) {
	var imageFile string
	if cfg.Output != "" {
		imageFile = cfg.Output
	} else {
		imageFile = filepath.Join(os.TempDir(), "image.png")
	}

	if err := saveImage(message.Content, imageFile); err != nil {
		log.Errorf("failed to save image: %v\n", err)
		return
	}

	if err := util.PrintImage(os.Stdout, imageFile); err != nil {
		if err := util.ViewImage(imageFile); err != nil {
			log.Errorf("failed to view image: %v\n", err)
		}
	}
}

func saveImage(b64Image string, dest string) error {
	// https://sw.kovidgoyal.net/kitty/graphics-protocol/
	img, _, err := image.Decode(base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64Image)))
	if err != nil {
		return err

	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}

	err = os.WriteFile(dest, buf.Bytes(), 0755)
	if err != nil {
		log.Errorf("failed to write image to %s: %v\n", dest, err)
	}
	log.Infof("Image content saved to %s\n", dest)

	return nil
}

// func agentCommandList() map[string]string {
// 	return AgentCommands
// }

func HandleCommand(cfg *internal.AppConfig) error {
	log.Debugf("HandleCommand: %s %v\n", cfg.Command, cfg.Args)

	cmd := cfg.Command

	if cmd != "" {
		// $ ai /command
		if strings.HasPrefix(cmd, "/") {
			name := strings.TrimSpace(cmd[1:])
			in, err := GetUserInput(cfg)
			if err != nil {
				return err
			}

			if name == "" && in.IsEmpty() {
				return internal.NewUserInputError("no command and message provided")
			}

			in.Agent = "script"
			in.Subcommand = name
			return RunSwarm(cfg, in)
		}

		// $ ai @agent
		if strings.HasPrefix(cmd, "@") {
			name := strings.TrimSpace(cmd[1:])
			if name == "" {
				// auto select
				// $ ai @ message...
				return AgentHelp(cfg)
			}

			// na := strings.SplitN(name, "/", 2)
			// agent := na[0]

			in, err := GetUserInput(cfg)
			if err != nil {
				return err
			}
			if in.IsEmpty() {
				return internal.NewUserInputError("no message content")
			}

			in.Agent = name

			// in.Agent = agent
			// if len(na) > 1 {
			// 	in.Subcommand = na[1]
			// }
			return RunSwarm(cfg, in)
		}
	}

	// auto select the best agent to handle the user query if there is message content
	// $ ai message...
	return AgentHelp(cfg)
}
