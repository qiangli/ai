package atm

import (
	"bytes"
	"maps"
	"slices"
	"strings"
	"text/template"

	"github.com/u-root/u-root/pkg/shlex"

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

var templateMimeTypes = []string{"text/x-go-template", "x-go-template", "template", "tpl"}

// check
// #! or // magic for large block of text
// {{ contained within for oneliner
func IsTemplate(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if strings.HasPrefix(s, "#!") || strings.HasPrefix(s, "//") {
		_, mime := ParseMimeType(s)
		return slices.Contains(templateMimeTypes, mime)
	}
	return strings.Contains(s, "{{")
}

// Check for mime-type specification.
// Returns content and mime type. remove first line for multiline text
func ParseMimeType(s string) (string, string) {
	var line string
	var data string
	parts := strings.SplitN(s, "\n", 2)
	// remove first line for multiline text
	if len(parts) == 2 {
		line = parts[0]
		data = parts[1]
	} else {
		line = parts[0]
		if len(line) > 256 {
			line = line[:256]
		}
		data = parts[0]
	}
	// shlex returns nil if not trimmed
	line = strings.TrimPrefix(line, "#!")
	opts := []string{"--mime-type", "--mime_type", "mime-type", "mime_type"}
	args := shlex.Argv(line)
	for i, v := range args {
		if slices.Contains(opts, v) {
			if len(args) > i+1 {
				return data, args[i+1]
			}
		}
		sa := strings.SplitN(v, "=", 2)
		if len(sa) == 1 {
			continue
		}
		if slices.Contains(opts, sa[0]) {
			return data, sa[1]
		}
	}
	return s, ""
}

// Check s for prefix "#!", "//" with mime-type specification
// or infix "{{" to apply template if found. otherise skip
func CheckApplyTemplate(tpl *template.Template, s string, data map[string]any) (string, error) {
	if strings.HasPrefix(s, "#!") || strings.HasPrefix(s, "//") {
		// parts := strings.SplitN(s, "\n", 2)
		// if len(parts) == 2 {
		// 	// remove hashbang line
		// 	// return applyTemplate(tpl, parts[1], data)
		// }
		// // remove leading hashbang
		// return applyTemplate(tpl, parts[0][2:], data)
		// not a template
		content, mime := ParseMimeType(s)
		if slices.Contains(templateMimeTypes, mime) {
			return applyTemplate(tpl, content, data)
		}
		return s, nil
	}
	// one liner - any string containing with {{
	if strings.Contains(s, "{{") {
		return applyTemplate(tpl, s, data)
	}
	// original
	return s, nil
}

// Merge and return the currently active args in the following order:
// + defaults from parameters
// + global envs
// + agent arguments
// + runtime args
func BuildEffectiveArgs(vars *api.Vars, agent *api.Agent, args map[string]any) map[string]any {
	var data = make(map[string]any)
	// defaults from agent parameters
	if agent != nil {
		if len(agent.Parameters) > 0 {
			obj := agent.Parameters["properties"]
			props, _ := api.ToMap(obj)
			for key, prop := range props {
				if p, ok := prop.(map[string]any); ok {
					if def, ok := p["default"]; ok {
						data[key] = def
					}
				}
			}
		}
	}
	// wont check vars.global - this should never be nil
	maps.Copy(data, vars.Global.GetAllEnvs())
	// agent arguments
	if agent != nil {
		maps.Copy(data, agent.Arguments)
	}
	maps.Copy(data, args)
	// predefined
	data["workspace"] = vars.RTE.Roots.Workspace.Path
	data["user"] = vars.RTE.User

	return data
}
