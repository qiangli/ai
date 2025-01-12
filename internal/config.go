package internal

import (
	_ "embed"
)

//go:embed ai.yaml
var configFileContent string

func GetDefaultConfig() string {
	return configFileContent
}

// global flags
var Debug bool // verbose output

var DryRun bool
var DryRunContent string

var WorkDir string
