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
