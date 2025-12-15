package atm

import (
	"bytes"
	"net/url"
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

func IsTemplate(s string) bool {
	// #! for large block of text
	// {{ for oneliner
	return strings.HasPrefix(s, "#!") || strings.Contains(s, "{{")
}

// Check s for prefix "#!" or "{{" to apply template if found. otherise skip
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
	return s, nil
}

// Load data from uri.
// Support file:// and data: protocols
func LoadURIContent(ws api.Workspace, uri string) (string, error) {
	var v string
	if strings.HasPrefix(uri, "data:") {
		return api.DecodeDataURL(v)
	} else {
		var f = uri
		if strings.HasPrefix(f, "file:") {
			v, err := url.Parse(f)
			if err != nil {
				return "", err
			}
			f = v.Path
		}
		data, err := ws.ReadFile(f, nil)
		if err != nil {
			return "", err
		}
		v = string(data)
	}

	return v, nil
}
