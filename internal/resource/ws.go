package resource

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed ws_base_system_role.md
var wsBaseSystemRoleTemplate string

//go:embed ws_base_user_role.md
var wsBaseUserRoleTemplate string

//go:embed ws_env_context.md
var wsEnvContext string

func GetWSBaseSystemRoleContent() string {
	return wsBaseSystemRoleTemplate
}

func GetWSBaseUserRoleContent(input string) (string, error) {
	tpl, err := template.New("userRole").Funcs(template.FuncMap{
		"maxLen": maxLen,
	}).Parse(wsBaseUserRoleTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]any{
		"input": input,
	}
	if err = tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type WSInput struct {
	Env          string
	HostDir      string
	ContainerDir string
	Input        string
}

func GetWSEnvContextInput(input *WSInput) (string, error) {
	tpl, err := template.New("userRole").Funcs(template.FuncMap{
		"maxLen": maxLen,
	}).Parse(wsEnvContext)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]any{
		"env":          input.Env,
		"hostDir":      input.HostDir,
		"containerDir": input.ContainerDir,
		"input":        input.Input,
	}
	if err = tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
