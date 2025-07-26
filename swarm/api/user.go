package api

import (
	"time"
)

type User struct {
	// uuid
	ID      string    `json:"id"`
	Created time.Time `json:"created"`

	// data
	Username string `json:"username"`
	Display  string `json:"display"`
}
