package resource

// import (
// 	"bytes"
// 	_ "embed"
// 	"text/template"
// )

// //go:embed sql_system_role.md
// var sqlSystemRoleTemplate string

// type DBInfo struct {
// 	Version     string
// 	ContextData string
// }

// func GetSqlSystemRoleContent(info *DBInfo) (string, error) {
// 	var tplOutput bytes.Buffer

// 	tpl, err := template.New("systemRole").Funcs(template.FuncMap{
// 		"maxLen": maxLen,
// 	}).Parse(sqlSystemRoleTemplate)
// 	if err != nil {
// 		return "", err
// 	}

// 	dialect, version := splitVersion(info.Version)
// 	data := map[string]any{
// 		"dialect": dialect,
// 		"version": version,
// 		"context": info.ContextData,
// 	}
// 	err = tpl.Execute(&tplOutput, data)
// 	if err != nil {
// 		return "", err
// 	}

// 	return tplOutput.String(), nil
// }
