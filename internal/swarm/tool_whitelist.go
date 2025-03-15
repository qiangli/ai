package swarm

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

func isAllowed(name string) bool {
	for _, v := range whitelist {
		if v == name {
			return true
		}
	}
	return false
}

func isDenied(name string) bool {
	for _, v := range denyList {
		if v == name {
			return true
		}
	}
	return false
}
