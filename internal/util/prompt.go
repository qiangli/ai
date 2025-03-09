package util

import (
	"bufio"
	"io"
	"strings"

	"github.com/qiangli/ai/internal/log"
)

func Confirm(ps string, choices []string, defaultChoice string, in io.Reader) (string, error) {
	memo := make(map[string]string)
	for _, v := range choices {
		choice := strings.ToLower(v)
		memo[choice] = choice
		memo[choice[:1]] = choice
	}

	reader := bufio.NewReader(in)
	for {
		log.Promptf(ps)

		resp, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		resp = strings.ToLower(strings.TrimSpace(resp))
		if resp == "" {
			return defaultChoice, nil
		}
		result, ok := memo[resp]
		if ok {
			return result, nil
		}
	}
}

func Prompt(ps string, in io.Reader) (string, error) {
	reader := bufio.NewReader(in)
	for {
		log.Promptf(ps)

		resp, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		resp = strings.ToLower(strings.TrimSpace(resp))
		if resp == "" {
			continue
		}
		return resp, nil
	}
}
