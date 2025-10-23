package util

import (
	"fmt"
	"os/user"
)

func WhoAmI() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to fetch username: %v", err)
	}
	return u.Username, nil
}
