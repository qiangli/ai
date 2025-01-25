package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Output format type
type outputValue string

func newOutputValue(val string, p *string) *outputValue {
	*p = val
	return (*outputValue)(p)
}
func (s *outputValue) Set(val string) error {
	for _, v := range []string{"raw", "markdown"} {
		if val == v {
			*s = outputValue(val)
			return nil
		}
	}
	return fmt.Errorf("invalid output format: %v. supported: raw, markdown", val)
}
func (s *outputValue) Type() string {
	return "string"
}
func (s *outputValue) String() string { return string(*s) }

// Template type
type templateValue string

func newTemplateValue(val string, p *string) *templateValue {
	*p = val
	return (*templateValue)(p)
}
func (s *templateValue) Set(val string) error {
	matches, err := filepath.Glob(val)
	if err != nil {
		return errors.New("error during file globbing")
	}
	if len(matches) != 1 {
		return errors.New("exactly one file must be provided")
	}

	fileInfo, err := os.Stat(matches[0])
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return errors.New("a file is required")
	}

	*s = templateValue(matches[0])
	return nil
}

func (s *templateValue) Type() string {
	return "string"
}

func (s *templateValue) String() string { return string(*s) }

// Files type
type filesValue struct {
	value   *[]string
	changed bool
}

func newFilesValue(val []string, p *[]string) *filesValue {
	ssv := new(filesValue)
	ssv.value = p
	*ssv.value = val
	return ssv
}

func (s *filesValue) Set(val string) error {
	matches, err := filepath.Glob(val)
	if err != nil {
		return fmt.Errorf("error processing glob pattern: %w", err)
	}

	if matches == nil {
		// no matches ignore
		return nil
	}

	if !s.changed {
		*s.value = matches
		s.changed = true
	} else {
		*s.value = append(*s.value, matches...)
	}
	return nil
}
func (s *filesValue) Append(val string) error {
	*s.value = append(*s.value, val)
	return nil
}

func (s *filesValue) Replace(val []string) error {
	out := make([]string, len(val))
	for i, d := range val {
		var err error
		out[i] = d
		if err != nil {
			return err
		}
	}
	*s.value = out
	return nil
}

func (s *filesValue) GetSlice() []string {
	out := make([]string, len(*s.value))
	if s.value != nil {
		copy(out, *s.value)
	}
	return out
}

func (s *filesValue) Type() string {
	return "string"
}

func (s *filesValue) String() string {
	if len(*s.value) == 0 {
		return ""
	}
	str, _ := s.writeAsCSV(*s.value)
	return "[" + str + "]"
}

func (s *filesValue) writeAsCSV(vals []string) (string, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	err := w.Write(vals)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n"), nil
}
