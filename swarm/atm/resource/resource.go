package resource

import (
	"embed"
)

//go:embed standard/*
var ResourceFS embed.FS

func NewStandardStore() *ResourceStore {
	return &ResourceStore{
		Base: "standard",
		FS:   ResourceFS,
	}
}

//go:embed shell_security_system.md
var ShellSecuritySystemRole string

//go:embed shell_security_user.md
var ShellSecurityUserRole string
