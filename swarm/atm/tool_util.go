package atm

import (
	"bytes"
	"maps"
	"slices"
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
		content, mime := api.ParseMimeType(s)
		if slices.Contains(api.TemplateMimeTypes, mime) {
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
func BuildEffectiveArgs(vars *api.Vars, agent *api.Agent, input map[string]any) map[string]any {
	if agent != nil {
		return BuildEffectiveParamArgs(vars, agent.Parameters, agent.Arguments, input)
	}
	return BuildEffectiveParamArgs(vars, nil, nil, input)
}

func BuildEffectiveParamArgs(vars *api.Vars, parameters api.Parameters, arguments api.Arguments, input map[string]any) map[string]any {
	var data = make(map[string]any)
	// // defaults from parameters
	// if len(parameters) > 0 {
	// 	// obj := parameters["properties"]
	// 	// props, _ := api.ToMap(obj)
	// 	// for key, prop := range props {
	// 	// 	if p, ok := prop.(map[string]any); ok {
	// 	// 		if def, ok := p["default"]; ok {
	// 	// 			data[key] = def
	// 	// 		}
	// 	// 	}
	// 	// }
	// 	maps.Copy(data, parameters.Defaults())
	// }
	// wont check vars.global - this should never be nil
	maps.Copy(data, vars.Global.GetAllEnvs())
	// arguments
	maps.Copy(data, arguments)

	maps.Copy(data, input)

	// predefined
	data["workspace"] = vars.Roots.Workspace.Path
	data["user"] = vars.User

	return data
}
