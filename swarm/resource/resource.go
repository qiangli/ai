package resource

import (
	"embed"
)

//go:embed standard/*
var ResourceFS embed.FS
