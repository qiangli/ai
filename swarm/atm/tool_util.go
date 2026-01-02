package atm

import (
	"bytes"
	"maps"
	"strings"
	"text/template"

	"github.com/qiangli/ai/swarm/api"
)

func applyTemplate(tpl *template.Template, text string, data any) (string, error) {
	t, err := tpl.Parse(text)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// crude check
// #! magic for large block of text
// {{ contained within for oneliner
func IsTemplate(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(s, "#!") || strings.Contains(s, "{{")
}

// Check s for prefix "#!" or infix "{{" to apply template if found. otherise skip
func CheckApplyTemplate(tpl *template.Template, s string, data map[string]any) (string, error) {
	if strings.HasPrefix(s, "#!") {
		// TODO parse the command line args?
		parts := strings.SplitN(s, "\n", 2)
		if len(parts) == 2 {
			// remove hashbang line
			return applyTemplate(tpl, parts[1], data)
		}
		// remove hashbang
		return applyTemplate(tpl, parts[0][2:], data)
	}
	// any string containing with {{
	if strings.Contains(s, "{{") {
		return applyTemplate(tpl, s, data)
	}
	// original
	return s, nil
}

// Merge and return the currently active args in the following order:
// + global envs
// + agent arguments if applicable
// + runtime args
func BuildEffectiveArgs(vars *api.Vars, agent *api.Agent, args map[string]any) map[string]any {
	var data = make(map[string]any)
	// wont check vars.global - this should never be nil
	maps.Copy(data, vars.Global.GetAllEnvs())
	if agent != nil {
		// defaults from parameters
		if len(agent.Parameters) > 0 {

		}
		maps.Copy(data, agent.Arguments)
	}
	maps.Copy(data, args)
	// predefined
	data["workspace"] = vars.RTE.Roots.Workspace
	data["user"] = vars.RTE.User

	return data
}
