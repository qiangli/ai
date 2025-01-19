package resource

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed meta_role.md
var metaRoleTemplate string

//go:embed shell_system_role.md
var shellSystemRoleTemplate string

//go:embed shell_security_role.md
var shellSecurityRoleTemplate string

//go:embed user_role.md
var userRoleTemplate string

//go:embed user_hint.txt
var userHint string

//go:embed user_input.txt
var userInputInstruction string

//go:embed user_example.txt
var userExample string

func GetMetaRoleContent() string {
	return metaRoleTemplate
}

func GetShellSystemRoleContent(info any) (string, error) {
	var tplOutput bytes.Buffer

	tpl, err := template.New("systemRole").Funcs(template.FuncMap{
		"maxLen": maxLen,
	}).Parse(shellSystemRoleTemplate)
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

func GetShellSecurityRoleContent() string {
	return shellSecurityRoleTemplate
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

func GetShellUserRoleContent(command string, message string) (string, error) {
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
