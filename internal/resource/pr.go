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

func GetPrDescriptionSystem() (string, error) {
	data := map[string]any{
		"schema":   prDecriptionSchema,
		"example":  prDescrptionExample,
		"maxFiles": 8,
	}
	return apply(prDescriptionSystem, data)
}

func GetPrReviewSystem() (string, error) {
	data := map[string]any{
		"schema":  prReviewSchema,
		"example": prReviewExample,
	}
	return apply(prReviewSystem, data)
}

func GetPrCodeSystem() (string, error) {
	data := map[string]any{
		"schema":         prCodeSchema,
		"example":        prCodeExample,
		"maxSuggestions": 8,
	}
	return apply(prCodeSystem, data)
}

func GetPrChangelogSystem() string {
	return prChangelogSystem
}

func FormatPrDescription(resp string) (string, error) {
	var data pr.PRDescription
	if err := tryUnmarshal(resp, &data); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}
	return apply(prDescriptionFormat, &data)
}

func FormatPrCodeSuggestion(resp string) (string, error) {
	var data pr.PRCodeSuggestions
	if err := tryUnmarshal(resp, &data); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}
	return apply(prCodeFormat, data.CodeSuggestions)
}

func FormatPrReview(resp string) (string, error) {
	var data pr.PRReview
	if err := tryUnmarshal(resp, &data); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}
	return apply(prReviewFormat, &data.Review)
}

func FormatPrChangelog(resp string) (string, error) {
	return apply(prChangelogFormat, &pr.PRChangelog{
		Changelog: resp,
		Today:     time.Now().Format("2006-01-02"),
	})
}
