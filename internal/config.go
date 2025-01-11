package internal

import (
	_ "embed"
)

//go:embed ai.yaml
var configFileContent string

func GetDefaultConfig() string {
	return configFileContent
}
