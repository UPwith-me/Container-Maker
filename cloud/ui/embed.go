package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distEmbed embed.FS

// DistDir returns the embedded dist directory
func DistDir() (fs.FS, error) {
	return fs.Sub(distEmbed, "dist")
}
