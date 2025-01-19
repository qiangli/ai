package resource

import (
	"testing"

	"github.com/qiangli/ai/internal/resource/pr"
)

func TestGetPrDescriptionSystem(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := GetPrDescriptionSystem()
			if err != nil {
				t.Errorf("GetPrDescriptionSystem: %v", err)
			}
			t.Logf("TestGetPrDescriptionSystem:\n%v", text)
		})
	}
}

func TestGetPrUser(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		diff string

		changelog string
	}{
		{
			name: "test1 - input",
			msg:  "my input",
			diff: "my diff",
		},
		{
			name: "test2 - no input",
			msg:  "",
			diff: "my diff",
		},
		{
			name:      "test3 - change log",
			msg:       "my input",
			diff:      "my diff",
			changelog: "my changelog",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txt, err := GetPrUser(&pr.Input{
				Instruction: tt.msg,
				Diff:        tt.diff,
				ChangeLog:   tt.changelog,
			})
			if err != nil {
				t.Errorf("TestGetPrUser: %v", err)
			}
			t.Logf("TestGetPrUser:\n%s\n%v", tt.name, txt)
		})
	}
}

func TestFormatPrDescription(t *testing.T) {
	out, err := FormatPrDescription(prDescrptionExample)
	if err != nil {
		t.Errorf("FormatPrDescription: %v", err)
	}
	t.Logf("TestFormatPrDescription:\n%v", out)
}

func TestGetPrReviewSystem(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := GetPrReviewSystem()
			if err != nil {
				t.Errorf("TestGetPrReviewSystem: %v", err)
			}
			t.Logf("TestGetPrReviewSystem:\n%v", text)
		})
	}
}

func TestGetPrCodeSystem(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := GetPrCodeSystem()
			if err != nil {
				t.Errorf("TestGetPrCodeSystem: %v", err)
			}
			t.Logf("TestGetPrCodeSystem:\n%v", text)
		})
	}
}
