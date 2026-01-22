package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Clip trims the string to the maxLen
func Clip(text string, maxLen int) string {
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}

// Head trims the string to the maxLen and replaces newlines with /.
func Head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "•")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// ParseDuration supports time.ParseDuration plus an extra suffix.
// Accepted formats include: '10' (seconds), '10s', '5m', '2h', '1d'.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("duration is required")
	}

	// Support day suffix like "2d".
	if strings.HasSuffix(s, "d") {
		v := strings.TrimSuffix(s, "d")
		if v == "" {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %v", s, err)
		}
		if f < 0 {
			return 0, fmt.Errorf("duration must be non-negative")
		}
		return time.Duration(f * float64(24*time.Hour)), nil
	}

	// If no unit is provided, treat as seconds (Unix sleep behavior).
	last := s[len(s)-1]
	if last >= '0' && last <= '9' {
		s = s + "s"
	}

	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, fmt.Errorf("duration must be non-negative")
	}
	return d, nil
}
