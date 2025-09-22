package watch

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gofrs/flock"

	"github.com/qiangli/ai/swarm/api"
)

func parseFile(path string, prefix string) (string, error) {
	// log.GetLogger(ctx).Debug("parseFile: %q\nprefix: %q\n", path, prefix)

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close file: %w", closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)

	readMulti := func() (string, error) {
		var lines []string
		for {
			if !scanner.Scan() {
				return "", fmt.Errorf("no line found")
			}
			line := scanner.Text()
			if strings.HasSuffix(line, "\\") {
				lines = append(lines, line[:len(line)-1]) // Remove the trailing backslash
				continue
			}
			lines = append(lines, line)
			return strings.TrimSpace(strings.Join(lines, "")), nil
		}
	}

	// read line and the next until the next line is empty
	readLine := func() (string, error) {
		var line, next string
		line, err = readMulti()
		if err != nil {
			return "", err
		}
		for {
			next, err = readMulti()
			if err != nil {
				return "", err
			}
			if len(next) == 0 {
				return line, nil
			}
			line = next
		}
	}

	// re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(prefix) + `\s*ai\s+.*`)
	re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(prefix) + `\s*(?i:(?:todo))\s+.*`)

	for {
		line, err := readLine()
		if err != nil {
			break
		}
		if len(prefix) == 0 || strings.HasPrefix(line, prefix) {
			if re.MatchString(line) {
				// log.GetLogger(ctx).Debug("found ai command: %s\n", line)
				return line, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return "", nil
}

func parseUserInput(line string, prefix string) (*api.UserInput, error) {
	// log.GetLogger(ctx).Debug("parseUserInput: %q\nprefix: %q\n", line, prefix)

	// re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(prefix) + `\s*ai\s+(.*)`)
	re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(prefix) + `\s*(?i:(?:todo))(?:\s+(.*))?$`)

	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("line does not match expected format")
	}

	line = strings.TrimSpace(matches[1])
	if len(line) == 0 {
		return &api.UserInput{}, nil
	}

	if line[0] == '@' {
		in := &api.UserInput{}
		parts := strings.SplitN(line, " ", 2)
		in.Agent = parts[0][1:]
		if len(parts) > 1 {
			in.Message = parts[1]
		}
		return in, nil
	}

	// if line[0] == '/' {
	// 	in := &api.UserInput{}
	// 	parts := strings.SplitN(line, " ", 2)
	// 	in.Agent = "script"
	// 	in.Command = parts[0]
	// 	if len(parts) > 1 {
	// 		in.Message = parts[1]
	// 	}
	// 	return in, nil
	// }

	return &api.UserInput{
		Message: line,
	}, nil
}

func replaceContentInFile(path, line string, prefix string, content string) error {
	fileLock := flock.New(path)

	locked, err := fileLock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return os.ErrExist
	}
	defer fileLock.Unlock()

	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result := strings.Replace(string(original), line+"\n", line+"\n"+prefix+"\n"+content+"\n", 1)

	return os.WriteFile(path, []byte(result), 0644)
}

func clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n..."
	}
	return text
}
