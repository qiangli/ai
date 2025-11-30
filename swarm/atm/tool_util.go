package atm

import (
	"bytes"
	"strings"
	"text/template"
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
	return strings.HasPrefix(s, "#!") || (strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}"))
}

func ApplyTemplate(tpl *template.Template, s string, data map[string]any) (string, error) {
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
	// any string starting with {{
	// a) to escape, add an additional char .e.g -
	// b) add {{""}} at the beginning to force applying
	if strings.HasPrefix(s, "{{") {
		return applyTemplate(tpl, s, data)
	}
	return s, nil
}
