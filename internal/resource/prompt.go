package resource

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed meta_role.md
var metaRoleTemplate string

//go:embed system_role.md
var systemRoleTemplate string

//go:embed user_role.md
var userRoleTemplate string

//go:embed user_hint.txt
var userHint string

//go:embed user_input.txt
var userInputInstruction string

//go:embed user_example.txt
var userExample string

//go:embed ai_help_role.md
var aiHelpRoleTemplate string

//go:embed ws_check_system_role.md
var wsCheckSystemRoleTemplate string

//go:embed ws_check_user_role.md
var wsCheckUserRoleTemplate string

//go:embed ws_user_input.md
var wsUserInputInstruction string

func GetMetaRoleContent() string {
	return metaRoleTemplate
}

func GetSystemRoleContent(info any) (string, error) {
	var tplOutput bytes.Buffer

	tpl, err := template.New("systemRole").Funcs(template.FuncMap{
		"maxLen": maxLen,
	}).Parse(systemRoleTemplate)
	if err != nil {
		return "", err
	}

	data := map[string]any{
		"info": info,
	}
	err = tpl.Execute(&tplOutput, data)
	if err != nil {
		return "", err
	}

	return tplOutput.String(), nil
}

func GetUserHint() string {
	return userHint
}

func GetUserExample() string {
	return userExample
}

func GetUserInputInstruction() string {
	return userInputInstruction
}

func GetUserRoleContent(command string, message string) (string, error) {
	tpl, err := template.New("userRole").Funcs(template.FuncMap{
		"maxLen": maxLen,
	}).Parse(userRoleTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]any{
		"command": command,
		"message": message,
	}
	if err = tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GetAIHelpRoleContent() string {
	return aiHelpRoleTemplate
}

func GetWSCheckSystemRoleContent() string {
	return wsCheckSystemRoleTemplate
}

func GetWSCheckUserRoleContent(input string) (string, error) {
	tpl, err := template.New("userRole").Funcs(template.FuncMap{
		"maxLen": maxLen,
	}).Parse(wsCheckUserRoleTemplate)
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

func GetWSUserInputInstruction(input *WSInput) (string, error) {
	tpl, err := template.New("userRole").Funcs(template.FuncMap{
		"maxLen": maxLen,
	}).Parse(wsUserInputInstruction)
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
