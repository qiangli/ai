package internal

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed resource/system_role.md
var systemRoleTemplate string

//go:embed resource/user_role.md
var userRoleTemplate string

//go:embed resource/user_hint.txt
var userHint string

//go:embed resource/user_input.txt
var userInputInstruction string

//go:embed resource/user_example.txt
var userExample string

func GetSystemRoleMessage() (string, error) {
	info, err := CollectSystemInfo()
	if err != nil {
		return "", err
	}
	var tplOutput bytes.Buffer

	tpl, err := template.New("systemRole").Funcs(template.FuncMap{
		"maxlen": MaxLen,
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

func GetUserRoleMessage(command string, message string) (string, error) {
	tpl, err := template.New("userRole").Funcs(template.FuncMap{
		"maxlen": MaxLen,
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
