//go:generate go run main.go

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func Generate() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	//
	dir := filepath.Dir(filename)
	parent := filepath.Dir(dir)

	outputFile, err := os.Create(filepath.Join(parent, "generated.go"))
	if err != nil {
		fmt.Println("Error creating prompts file:", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	generatePrompts(outputFile, dir)
	generateAgentCommandMap(outputFile, parent)
	// generateAgentConfigMap(outputFile, parent)
}

const generated = `//DO NOT EDIT. This file is generated.
package resource

import _ "embed"

%s

var Prompts = map[string]string{
%s
}

//go:embed common.yaml
var CommonData []byte

type AgentConfig struct {
	Name        string
	Description string
	Overview    string
	Internal    bool
	Data		[]byte
}

`

func generatePrompts(outputFile *os.File, dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		os.Exit(1)
	}

	base := filepath.Base(dir)

	var embeds []string
	var entries []string

	for _, file := range files {
		if !file.IsDir() {
			fileName := file.Name()
			if filepath.Ext(fileName) == ".go" {
				continue
			}
			fileBase := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			embeds = append(embeds, fmt.Sprintf("//go:embed %s/%s", base, fileName))
			embeds = append(embeds, fmt.Sprintf("var %s string", fileBase))
			embeds = append(embeds, "")

			entries = append(entries, fmt.Sprintf("\t\"%s\": %s,", fileBase, fileBase))
		}
	}

	fmt.Fprintln(outputFile, fmt.Sprintf(generated, strings.Join(embeds, "\n"), strings.Join(entries, "\n")))
}

type AgentsConfig struct {
	Agents []AgentConfig `yaml:"agents"`
}

type AgentConfig struct {
	Name string `yaml:"name"`

	Description string `yaml:"description"`
	Overview    string `yaml:"overview"`
	Internal    bool   `yaml:"internal"`

	Resource string `yaml:"-"`
	Data     []byte `yaml:"-"`
}

var agentMap map[string]AgentConfig

func generateAgentCommandMap(outfile *os.File, dir string) {
	agentMap = make(map[string]AgentConfig)
	err := loadAgentsFromDirectory(dir)
	if err != nil {
		log.Fatalf("Error loading agents: %v", err)
	}

	keys := make([]string, 0, len(agentMap))
	for k := range agentMap {
		keys = append(keys, k)
	}
	// Sort the keys
	sort.Strings(keys)
	var embeds []string
	var entries []string

	resourceMap := make(map[string]string)
	for _, name := range keys {
		config := agentMap[name]
		resourceName := strings.ReplaceAll(config.Resource, "/", "_")
		resourceName = strings.ReplaceAll(resourceName, "-", "_")
		resourceName = strings.ReplaceAll(resourceName, "-", "_")
		resourceName = strings.ReplaceAll(resourceName, ".", "_")
		if _, ok := resourceMap[config.Resource]; ok {
			continue
		}
		resourceMap[config.Resource] = resourceName
		embeds = append(embeds, fmt.Sprintf("//go:embed %s", config.Resource))
		embeds = append(embeds, fmt.Sprintf("var %s_data []byte", resourceName))
		embeds = append(embeds, "")
	}

	for _, name := range keys {
		config := agentMap[name]
		resourceName := resourceMap[config.Resource]
		entry := fmt.Sprintf("\t\"%s\": {\nName: \"%s\",\nDescription: \"%s\",\nInternal: %t,\nData: %s,\nOverview: \"%s\",\n},",
			name,
			config.Name,
			strings.Replace(config.Description, `"`, `\"`, -1),
			config.Internal,
			resourceName+"_data",
			strings.Replace(config.Overview, `"`, `\"`, -1),
		)
		entries = append(entries, entry)
	}

	fmt.Fprintf(outfile, "\n%s\nvar AgentCommandMap = map[string]AgentConfig {\n%s\n}\n", strings.Join(embeds, "\n"), strings.Join(entries, "\n"))
}

func loadAgentsFromDirectory(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			err = loadAgentsFromFile(root, path)
			if err != nil {
				log.Printf("Failed to load file %s: %v", path, err)
			}
		}
		return nil
	})
}

func loadAgentsFromFile(root, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg AgentsConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		fmt.Println("Error getting relative path:", err)
		os.Exit(1)
	}
	for _, agent := range cfg.Agents {
		agent.Resource = rel
		agentMap[agent.Name] = agent
	}
	return nil
}

func main() {
	Generate()
	fmt.Println("Resource mapping generated")
}
