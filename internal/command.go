package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qiangli/ai/cli/internal/log"
)

// ListCommands returns the full path of the first valid executable command encountered in the PATH
func ListCommands(nameOnly bool) ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil, errors.New("PATH environment variable is not set")
	}

	uniqueCommands := make(map[string]string) // command name -> full path
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, pathDir := range paths {
		files, err := os.ReadDir(pathDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			commandName := file.Name()
			fullPath := filepath.Join(pathDir, commandName)

			// Check if the file is executable and the command hasn't been registered yet
			if !file.IsDir() && IsExecutable(fullPath) {
				if _, exists := uniqueCommands[commandName]; !exists {
					uniqueCommands[commandName] = fullPath
				}
			}
		}
	}

	commands := make([]string, 0, len(uniqueCommands))
	for name, fullPath := range uniqueCommands {
		if nameOnly {
			commands = append(commands, name)
			continue
		}
		commands = append(commands, fullPath)
	}

	return commands, nil
}

func IsExecutable(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode.IsRegular() && mode&0111 != 0
}

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

func AgentCommand(cfg *Config, args []string) error {
	log.Debugf("Agent command: %v\n", args)
	if at := args[0]; strings.HasPrefix(at, "@") {
		name := strings.TrimSpace(at[1:])
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

		msg, err := GetUserInput(cfg, args)
		if err != nil {
			return err
		}
		if msg == "" {
			return NewUserInputError("no message content")
		}

		switch name {
		case "ask":
			agent, err := NewChat(cfg)
			if err != nil {
				return err
			}
			ctx := context.TODO()
			resp, err := agent.Send(ctx, msg)
			if err != nil {
				return err
			}
			processContent(cfg, resp.Content)
		default:
		}

		log.Debugf("agent command completed: %s message: %s\n", args[0], msg)
		return nil
	}
	return fmt.Errorf("invalid command: %s", args[0])
}

func SlashCommand(cfg *Config, args []string) error {
	log.Debugf("Slash command: %v\n", args)

	if slash := args[0]; strings.HasPrefix(slash, "/") {
		name := strings.TrimSpace(slash[1:])
		if name != "" {
			name = filepath.Base(name)
		}

		msg, err := GetUserInput(cfg, args)
		if err != nil {
			return err
		}

		if name == "" && msg == "" {
			return NewUserInputError("no command and message provided")
		}

		agent, err := NewScriptAgent(cfg)
		if err != nil {
			return err
		}

		if cfg.DryRun {
			log.Infof("Dry run mode. No API call will be made!\n")
			if cfg.DryRunFile != "" {
				log.Infof("Content of %s will be returned.\n", cfg.DryRunFile)
			}
		}

		log.Infoln("Sending request to the AI model...")
		ctx := context.TODO()
		resp, err := agent.Send(ctx, name, msg)
		if err != nil {
			return err
		}
		processContent(cfg, resp.Content)

		log.Debugf("Slash command completed: %s message: %s\n", args[0], msg)
		return nil
	}
	return fmt.Errorf("invalid command: %s", args[0])
}

func InfoCommand(cfg *Config, args []string) error {
	info, err := collectSystemInfo()
	if err != nil {
		log.Errorln(err)
		return err
	}
	log.Infoln(info)
	return nil
}

func ListCommand(cfg *Config, args []string) error {
	var nameOnly bool
	if len(args) == 2 && args[1] == "--name" {
		nameOnly = true
	}

	list, err := ListCommands(nameOnly)
	if err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}

	log.Printf("Available commands on the system:\n\n")
	sort.Strings(list)
	for _, c := range list {
		log.Println(c)
	}
	log.Printf("\nTotal: %v\n", len(list))
	return nil
}

func HelpCommand(cfg *Config, args []string) error {
	log.Println("AI Command Line Tool\n")
	log.Println("Usage:")
	log.Println("  ai [OPTIONS] COMMAND [message...]\n")
	hint := GetUserHint()
	log.Println(hint)
	ex := GetUserInputInstruction()
	log.Println(ex)
	return nil
}

func collectSystemInfo() (string, error) {
	info, err := CollectSystemInfo()
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
	log.Println(content)

	doc := ParseMarkdown(content)
	total := len(doc.CodeBlocks)
	if total > 0 {
		log.Printf("\n=== CODE BLOCKS (%v) ===\n", total)
		for i, v := range doc.CodeBlocks {
			log.Printf("\n===\n%s\n=== %v/%v ===\n", v.Code, i+1, total)
			ProcessBashScript(cfg, v.Code)
		}
		log.Println("=== END ===\n")
	}
}
