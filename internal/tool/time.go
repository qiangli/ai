package tool

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const worldTimeUrl = "https://worldtimeapi.org/api/ip"

func WhatTimeIsIt() (string, error) {
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
