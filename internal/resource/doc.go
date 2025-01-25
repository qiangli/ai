package resource

import (
	_ "embed"
)

//go:embed doc/compose_system.md
var docComposeSystem string

type DocCompose struct {
	Template string
	Draft    string
}

func GetDocComposeSystem(input *DocCompose) (string, error) {
	data := map[string]any{
		"template": input.Template,
		"draft":    input.Draft,
	}
	return apply(docComposeSystem, data)
}
