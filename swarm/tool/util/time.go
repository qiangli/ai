package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const worldTimeUrl = "https://worldtimeapi.org/api/ip"

func WorldTime() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	response, err := client.Get(worldTimeUrl)
	if err != nil {
		return "", fmt.Errorf("failed to connect: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch time, status code: %d", response.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("unexpected issue - %v", err)
	}

	datetime, ok := result["datetime"]
	if !ok {
		return "", fmt.Errorf("'datetime' field not found in response")
	}

	return datetime.(string), nil
}

// GetLocalTZ tries to load a time.Location by name, or returns the system's local time zone.
// If both fail, returns an error.
func GetLocalTZ(localTZOverride string) (*time.Location, error) {
	if localTZOverride != "" {
		loc, err := time.LoadLocation(localTZOverride)
		if err != nil {
			return nil, err
		}
		return loc, nil
	}

	loc := time.Now().Location()
	if loc == nil {
		return nil, errors.New("could not determine local timezone - Location is nil")
	}
	return loc, nil
}

type TimeResult struct {
	Timezone string `json:"timezone"`
	Datetime string `json:"datetime"`
	IsDST    bool   `json:"is_dst"`
}

func (t TimeResult) String() string {
	b, err := json.Marshal(t)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func GetCurrentTime(timezoneName string) (*TimeResult, error) {
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return nil, err
	}

	// Get current time in specified timezone
	currentTime := time.Now().In(location)

	result := &TimeResult{
		Timezone: timezoneName,
		Datetime: currentTime.Format("2006-01-02T15:04:05"), // ISO-like, seconds precision
		IsDST:    currentTime.IsDST(),
	}

	return result, nil
}

type TimeConversionResult struct {
	Source         TimeResult `json:"source_timezone"`
	Target         TimeResult `json:"target_timezone"`
	TimeDifference string     `json:"time_difference"`
}

func (t TimeConversionResult) String() string {
	b, err := json.Marshal(t)
	if err != nil {
		return "<error marshalling TimeConversionResult>"
	}
	return string(b)
}

func ConvertTime(sourceTz, timeStr, targetTz string) (TimeConversionResult, error) {
	// Load source and target location
	sourceLocation, err := time.LoadLocation(sourceTz)
	if err != nil {
		return TimeConversionResult{}, fmt.Errorf("invalid source timezone: %v", err)
	}
	targetLocation, err := time.LoadLocation(targetTz)
	if err != nil {
		return TimeConversionResult{}, fmt.Errorf("invalid target timezone: %v", err)
	}

	// Parse the time string in HH:MM 24-hour format
	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return TimeConversionResult{}, fmt.Errorf("invalid time format. Expected HH:MM [24-hour format]")
	}

	// Use today's date
	now := time.Now().In(sourceLocation)
	sourceTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		parsedTime.Hour(), parsedTime.Minute(), 0, 0,
		sourceLocation,
	)

	// Convert to target timezone
	targetTime := sourceTime.In(targetLocation)

	// Calculate offset difference
	_, sourceOffset := sourceTime.Zone()
	_, targetOffset := targetTime.Zone()
	hoursDifference := float64(targetOffset-sourceOffset) / 3600

	var timeDiffStr string
	if hoursDifference == float64(int(hoursDifference)) {
		timeDiffStr = fmt.Sprintf("%+.1fh", hoursDifference)
	} else {
		// Keep at most 2 decimal places, remove trailing zeros/dots
		timeDiffStr = fmt.Sprintf("%+.2f", hoursDifference)
		for strings.HasSuffix(timeDiffStr, "0") {
			timeDiffStr = timeDiffStr[:len(timeDiffStr)-1]
		}
		timeDiffStr = strings.TrimSuffix(timeDiffStr, ".")
		timeDiffStr += "h"
	}

	return TimeConversionResult{
		Source: TimeResult{
			Timezone: sourceTz,
			Datetime: sourceTime.Format(time.RFC3339),
			IsDST:    sourceTime.IsDST(),
		},
		Target: TimeResult{
			Timezone: targetTz,
			Datetime: targetTime.Format(time.RFC3339),
			IsDST:    targetTime.IsDST(),
		},
		TimeDifference: timeDiffStr,
	}, nil
}
