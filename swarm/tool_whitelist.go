package swarm

import (
	"slices"
	"strings"
)

// always allowed
var toolsList = []string{
	// core
	"man",
	"col",
	"command",
	"which",
	"ls",
	"test",
	//
	// TODO yaml config
	"git",
	"zstd",
	"unzstd",
	"tar",
	"unzip",
	"zip",
	"gzip",
	"gunzip",
	"curl",
	"wget",
}

// TODO Read from config
var allowList = []string{}

// TODO Read from config
var denyList = []string{
	"rm",
	"sudo",
}

func isAllowed(allowed []string, command string) bool {
	name := strings.TrimSpace(strings.SplitN(command, " ", 2)[0])
	if allowed == nil {
		allowed = allowList
	}
	var whitelist = append(toolsList, allowed...)
	return slices.Contains(whitelist, name)
}

func isDenied(denied []string, command string) bool {
	name := strings.TrimSpace(strings.SplitN(command, " ", 2)[0])
	if denied == nil {
		denied = denyList
	}
	return slices.Contains(denied, name)
}
