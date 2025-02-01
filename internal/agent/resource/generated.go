// DO NOT EDIT. This file is generated.
package resource

import _ "embed"

//go:embed prompts/agent_meta_system_role.md
var agent_meta_system_role string

//go:embed prompts/agent_sub_system_role.md
var agent_sub_system_role string

//go:embed prompts/git_sub_system_role.md
var git_sub_system_role string

//go:embed prompts/gptr_sub_system_role.md
var gptr_sub_system_role string

//go:embed prompts/pr_sub_system_role.md
var pr_sub_system_role string

var Prompts = map[string]string{
	"agent_meta_system_role": agent_meta_system_role,
	"agent_sub_system_role":  agent_sub_system_role,
	"git_sub_system_role":    git_sub_system_role,
	"gptr_sub_system_role":   gptr_sub_system_role,
	"pr_sub_system_role":     pr_sub_system_role,
}
