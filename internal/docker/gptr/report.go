package gptr

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/log"
)

// https://docs.gptr.dev/docs/gpt-researcher/getting-started/cli

var ReportTypes = map[string]string{
	"research_report": "Summary - Short and fast (~2 min)",
	"detailed_report": "Detailed - In depth and longer (~5 min)",
	"resource_report": "Resource Report",
	"outline_report":  "Outline Report",
	"custom_report":   "Custom Report",
	"subtopic_report": "Subtopic Report",
}

var Tones = map[string]string{
	"objective":   "Impartial and unbiased presentation",
	"formal":      "Academic standards with sophisticated language",
	"analytical":  "Critical evaluation and examination",
	"persuasive":  "Convincing viewpoint",
	"informative": "Clear and comprehensive information",
	"explanatory": "Clarifying complex concepts",
	"descriptive": "Detailed depiction",
	"critical":    "Judging validity and relevance",
	"comparative": "Juxtaposing different theories",
	"speculative": "Exploring hypotheses",
	"reflective":  "Personal insights",
	"narrative":   "Story-based presentation",
	"humorous":    "Light-hearted and engaging",
	"optimistic":  "Highlighting positive aspects",
	"pessimistic": "Focusing on challenges",
}

type ReportArgs struct {
	ReportType string `json:"report_type"`
	Tone       string `json:"tone"`
}

func GenerateReport(ctx context.Context, reportType, tone, input string, out string) error {
	if len(reportType) == 0 {
		reportType = "research_report"
	}

	if len(tone) == 0 {
		tone = "objective"
	}

	query := strings.TrimSpace(input)
	if len(query) == 0 {
		return fmt.Errorf("query is required")
	}

	log.Infoln("Building gptr docker image, please wait...")
	if err := BuildImage(ctx); err != nil {
		return err
	}

	log.Infof("Generating report type: %s tone: %s...\n", reportType, tone)
	if err := RunContainer(ctx, reportType, tone, query, out); err != nil {
		return err
	}
	return nil
}

func ToReportArgs(sub string) *ReportArgs {
	reportType, tone := SplitSub(sub)
	return &ReportArgs{
		ReportType: reportType,
		Tone:       tone,
	}
}

func SplitSub(s string) (string, string) {
	var reportType string
	var tone string

	s = strings.Trim(s, "/")
	parts := strings.SplitN(s, "/", 2)

	if len(parts) == 1 {
		reportType = parts[0]
	}
	if len(parts) == 2 {
		reportType = parts[0]
		tone = parts[1]
	}

	validType := func() bool {
		for k := range ReportTypes {
			if reportType == k {
				return true
			}
		}
		return false
	}
	validTone := func() bool {
		for k := range Tones {
			if tone == k {
				return true
			}
		}
		return false
	}
	if !validType() {
		reportType = "research_report"
	}
	if !validTone() {
		tone = "objective"
	}
	return reportType, tone
}
