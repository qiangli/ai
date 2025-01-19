package resource

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"text/template"
	"time"

	"github.com/qiangli/ai/internal/resource/pr"
)

//go:embed pr/user.md
var prUser string

//go:embed pr/description_system.md
var prDescriptionSystem string

//go:embed pr/description_schema.json
var prDecriptionSchema string

//go:embed pr/description_example.json
var prDescrptionExample string

//go:embed pr/description_format.md
var prDescriptionFormat string

//go:embed pr/review_system.md
var prReviewSystem string

//go:embed pr/review_schema.json
var prReviewSchema string

//go:embed pr/review_example.json
var prReviewExample string

//go:embed pr/review_format.md
var prReviewFormat string

//go:embed pr/code_system.md
var prCodeSystem string

//go:embed pr/code_schema.json
var prCodeSchema string

//go:embed pr/code_example.json
var prCodeExample string

//go:embed pr/code_format.md
var prCodeFormat string

//go:embed pr/changelog_system.md
var prChangelogSystem string

func GetPrUser(in *pr.Input) (string, error) {
	tpl, err := template.New("prDescriptionUser").Funcs(tplFuncMap).Parse(prUser)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]any{
		"instruction": in.Instruction,
		"diff":        in.Diff,
		"changelog":   in.ChangeLog,
		"today":       time.Now().Format("2006-01-02"),
	}
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetPrDescriptionSystem() (string, error) {
	tpl, err := template.New("prDescriptionSystem").Parse(prDescriptionSystem)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]any{
		"schema":   prDecriptionSchema,
		"example":  prDescrptionExample,
		"maxFiles": 8,
	}
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func formatPr(format string, data any) (string, error) {
	tpl, err := template.New("prFormat").Funcs(tplFuncMap).Parse(format)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetPrReviewSystem() (string, error) {
	tpl, err := template.New("prReviewSystem").Parse(prReviewSystem)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]any{
		"schema":  prReviewSchema,
		"example": prReviewExample,
	}
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetPrCodeSystem() (string, error) {
	tpl, err := template.New("prCodeSystem").Parse(prCodeSystem)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]any{
		"schema":         prCodeSchema,
		"example":        prCodeExample,
		"maxSuggestions": 8,
	}
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetPrChangelogSystem() string {
	return prChangelogSystem
}

func FormatPrDescription(resp string) (string, error) {
	var data pr.PRDescription
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return "", err
	}
	return formatPr(prDescriptionFormat, &data)
}

func FormatPrCodeSuggestion(resp string) (string, error) {
	var data pr.PRCodeSuggestions
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return "", err
	}
	return formatPr(prCodeFormat, &data)
}

func FormatPrReview(resp string) (string, error) {
	var data pr.PRReview
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return "", err
	}
	return formatPr(prReviewFormat, &data)
}
