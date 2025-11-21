package swarm

import (
	"bytes"
	"fmt"
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

func isTemplate(s string) bool {
	return strings.HasPrefix(s, "#!") || (strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}"))
}

func applyGlobal(tpl *template.Template, s string, env map[string]any) (string, error) {
	if strings.HasPrefix(s, "#!") {
		// TODO parse the command line args?
		parts := strings.SplitN(s, "\n", 2)
		if len(parts) == 2 {
			// remove hashbang line
			return applyTemplate(tpl, parts[1], env)
		}
		// remove hashbang
		return applyTemplate(tpl, parts[0][2:], env)
	}
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		return applyTemplate(tpl, s, env)
	}
	return s, nil
}

func ToResult(data any) *api.Result {
	if v, ok := data.(*api.Result); ok {
		if len(v.Content) == 0 {
			return v
		}
		if v.MimeType == api.ContentTypeImageB64 {
			return v
		}
		if strings.HasPrefix(v.MimeType, "text/") {
			return &api.Result{
				MimeType: v.MimeType,
				Value:    string(v.Content),
			}
		}
		return &api.Result{
			MimeType: v.MimeType,
			Value:    dataURL(v.MimeType, v.Content),
		}
	}
	if s, ok := data.(string); ok {
		return &api.Result{
			Value: s,
		}
	}
	return &api.Result{
		Value: fmt.Sprintf("%v", data),
	}
}
