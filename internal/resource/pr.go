package resource

import (
	_ "embed"
	"fmt"
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

//go:embed pr/changelog_format.md
var prChangelogFormat string

func GetPrUser(in *pr.Input) (string, error) {
	data := map[string]any{
		"instruction": in.Instruction,
		"diff":        in.Diff,
		"changelog":   in.ChangeLog,
		"today":       time.Now().Format("2006-01-02"),
	}
	return apply(prUser, data)
}

func getPrDescriptionSystem() (string, error) {
	data := map[string]any{
		"schema":   prDecriptionSchema,
		"example":  prDescrptionExample,
		"maxFiles": 8,
	}
	return apply(prDescriptionSystem, data)
}

func getPrReviewSystem() (string, error) {
	data := map[string]any{
		"schema":  prReviewSchema,
		"example": prReviewExample,
	}
	return apply(prReviewSystem, data)
}

func getPrCodeSystem() (string, error) {
	data := map[string]any{
		"schema":         prCodeSchema,
		"example":        prCodeExample,
		"maxSuggestions": 8,
	}
	return apply(prCodeSystem, data)
}

func getPrChangelogSystem() string {
	return prChangelogSystem
}

func GetPrSystem(sub string) (string, error) {
	switch sub {
	case "describe":
		return getPrDescriptionSystem()
	case "review":
		return getPrReviewSystem()
	case "improve":
		return getPrCodeSystem()
	case "changelog":
		return getPrChangelogSystem(), nil
	}
	return "", fmt.Errorf("unknown @pr subcommand: %s", sub)
}

func formatPrDescription(resp string) (string, error) {
	var data pr.PRDescription
	if err := tryUnmarshal(resp, &data); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}
	return apply(prDescriptionFormat, &data)
}

func formatPrCodeSuggestion(resp string) (string, error) {
	var data pr.PRCodeSuggestions
	if err := tryUnmarshal(resp, &data); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}
	return apply(prCodeFormat, data.CodeSuggestions)
}

func formatPrReview(resp string) (string, error) {
	var data pr.PRReview
	if err := tryUnmarshal(resp, &data); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}
	return apply(prReviewFormat, &data.Review)
}

func formatPrChangelog(resp string) (string, error) {
	return apply(prChangelogFormat, &pr.PRChangelog{
		Changelog: resp,
		Today:     time.Now().Format("2006-01-02"),
	})
}

func FormatPrResponse(sub, resp string) (string, error) {
	switch sub {
	case "describe":
		return formatPrDescription(resp)
	case "review":
		return formatPrReview(resp)
	case "improve":
		return formatPrCodeSuggestion(resp)
	case "changelog":
		return formatPrChangelog(resp)
	}
	return "", fmt.Errorf("unknown @pr subcommand: %s", sub)
}
