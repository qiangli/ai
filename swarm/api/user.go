package api

// import (
// 	"time"
// )

type User struct {
	// uuid
	ID string `json:"id"`

	//
	// Username string `json:"username"`
	// emoji + nickname
	Display string `json:"display"`

	Email string `json:"email"`
	// full/first,last
	Name   string `json:"name"`
	Avatar string `json:"avatar"`

	// Created time.Time `json:"created"`
}
