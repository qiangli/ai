package internal

import (
	"fmt"
	"sort"
	"strings"
)

var availableAgents = map[string]string{
	"ask": "Ask a general question",
	// "aider":      "AI pair programming in your terminal",
	// "openhands":  "A platform for software development agents powered by AI",
	// "vanna":      "Let Vanna.AI write your SQL for you",
	// "research": "Autonomous agent designed for comprehensive web and local research",
}

func ListAgents() (map[string]string, error) {
	return availableAgents, nil
}

func AvailableAgents() string {
	dict, _ := ListAgents()
	list := make([]string, 0, len(dict))
	for k, v := range dict {
		list = append(list, fmt.Sprintf("%s: %s", k, v))
	}
	sort.Strings(list)
	var result strings.Builder
	for _, v := range list {
		result.WriteString(fmt.Sprintf("%s\n", v))
	}
	return result.String()
}
