package resource

import (
	_ "embed"

	"github.com/qiangli/ai/internal/resource/cli"
)

type AgentDetect struct {
	Agent   string `json:"agent"`
	Command string `json:"command"`
}

type ConfigSchema = cli.ConfigSchema

//go:embed cli/config_system.md
var cliConfigSystem string

//go:embed cli/config_schema.json
var cliConfigSchema string

//go:embed cli/config_user.md
var cliConfigUser string

//go:embed cli/agent_detect_system.md
var cliAgentDetectSystem string

//go:embed cli/pr_sub_system.md
var cliPrSubSystem string

func GetCliConfigSystem() (string, error) {
	data := map[string]any{
		"schema": cliConfigSchema,
	}
	return apply(cliConfigSystem, data)
}

func GetCliConfigUser(input string) (string, error) {
	data := map[string]any{
		"input": input,
	}
	return apply(cliConfigUser, data)
}

func GetCliAgentDetectSystem() string {
	return cliAgentDetectSystem
}

func GetCliPrSubSystem() string {
	return cliPrSubSystem
}

//go:embed cli/gptr_report_system.md
var cliGptrReportSystem string

func GetCliGptrReportSystem(reportTypes, tones map[string]string) (string, error) {
	return apply(cliGptrReportSystem, map[string]any{
		"ReportTypes": reportTypes,
		"Tones":       tones,
	})
}
