package api

type ContextKey string

const SwarmUserContextKey ContextKey = "swarm_user"

// const defaultAgent = ""

type SessionID string

type User struct {
	// uuid
	ID string `json:"id"`

	// emoji + nickname
	Display string `json:"display"`

	Email string `json:"email"`

	// full/first,last
	Name string `json:"name"`

	Avatar string `json:"avatar"`

	Settings map[string]any `json:"settings"`
}

// func (r *User) Agent() string {
// 	if r.Settings == nil {
// 		return defaultAgent
// 	}
// 	agent := r.Settings["agent"]
// 	if agent != "" {
// 		if s, ok := agent.(string); ok {
// 			return s
// 		}
// 	}
// 	return defaultAgent
// }

// func (r *User) SetAgent(s string) {
// 	r.Settings["agent"] = s
// }
