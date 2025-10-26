package api

type ContextKey string

const SwarmUserContextKey ContextKey = "swarm_user"

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
}
