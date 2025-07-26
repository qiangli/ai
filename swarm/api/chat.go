package api

import (
	"time"
)

type Chat struct {
	// uuid
	ID      string    `json:"id"`
	UserID  string    `json:"userId"`
	Created time.Time `json:"created"`

	// data
	Title string `json:"title"`
}
