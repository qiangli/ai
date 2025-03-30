package swarm

import (
	"slices"
	"strings"
)

// always allowed
var toolsList = []string{
	"man",
	"col",
	"command",
	"which",
	"ls",
	"test",
	//
	// TODO git mcp instead?
	"git",
}

// Read from config
var allowList = []string{}

// Read from config
var denyList = []string{
	"env",
	"printenv",
	"rm",
}

var whitelist = append(toolsList, allowList...)

func isAllowed(command string) bool {
	name := strings.TrimSpace(strings.SplitN(command, " ", 2)[0])
	return slices.Contains(whitelist, name)
}

func isDenied(command string) bool {
	name := strings.TrimSpace(strings.SplitN(command, " ", 2)[0])
	return slices.Contains(denyList, name)
}
