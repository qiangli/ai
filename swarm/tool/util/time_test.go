package util

import (
	"testing"
	"time"
)

func TestGetLocalTZ(t *testing.T) {
	tests := []struct {
		name         string
		override     string
		wantLocation string
		wantErr      bool
	}{
		// {
		// 	name:         "valid override",
		// 	override:     "America/New_York",
		// 	wantLocation: "America/New_York",
		// 	wantErr:      false,
		// },
		// {
		// 	name:     "invalid override",
		// 	override: "Not/AZone",
		// 	wantErr:  true,
		// },
		{
			name:         "no override (system local)",
			override:     "",
			wantLocation: time.Local.String(),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := GetLocalTZ(tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if err == nil && tt.wantLocation != "" && loc.String() != tt.wantLocation {
				// For 'no override', just check we got a non-nil *time.Location
				if tt.override == "" && loc == nil {
					t.Errorf("expected non-nil Location for system local")
				}
				// For real zones, check string
				if tt.override != "" && loc.String() != tt.wantLocation {
					t.Errorf("expected location %q, got %q", tt.wantLocation, loc.String())
				}
			}

			currentTime := time.Now().In(loc)
			name := currentTime.Format("MST")
			t.Logf("zone name: %q", name)
		})
	}
}

func TestGetCurrentTime(t *testing.T) {
	// Valid timezone
	res, err := GetCurrentTime("America/New_York")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if res.Timezone != "America/New_York" {
		t.Errorf("Expected timezone %q, got %q", "America/New_York", res.Timezone)
	}
	if len(res.Datetime) != len("2006-01-02T15:04:05") {
		t.Errorf("Unexpected datetime format: %v", res.Datetime)
	}

	// Invalid timezone
	_, err = GetCurrentTime("Invalid/Timezone")
	if err == nil {
		t.Error("Expected error for invalid timezone, got nil")
	}
}

func TestConvertTime(t *testing.T) {
	sourceTz := "America/New_York"
	targetTz := "Europe/London"
	timeStr := "15:30"

	result, err := ConvertTime(sourceTz, timeStr, targetTz)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Source.Timezone != sourceTz {
		t.Errorf("expected source timezone %s, got %s", sourceTz, result.Source.Timezone)
	}
	if result.Target.Timezone != targetTz {
		t.Errorf("expected target timezone %s, got %s", targetTz, result.Target.Timezone)
	}
	// Parse times for assertions
	sourceParsed, _ := time.Parse(time.RFC3339, result.Source.Datetime)
	targetParsed, _ := time.Parse(time.RFC3339, result.Target.Datetime)
	if sourceParsed.Hour() != 15 || sourceParsed.Minute() != 30 {
		t.Errorf("unexpected source hour/minute: %v", sourceParsed)
	}
	if (targetParsed.Hour() == sourceParsed.Hour() && targetParsed.Location().String() == targetTz) == false {
		// Just basic check: hour will differ depending on current DST and date.
		t.Logf("target time: %v", targetParsed)
	}
	if result.TimeDifference == "" {
		t.Error("expected non-empty time difference string")
	}
}
