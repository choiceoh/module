package main

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionRaw string

func appVersion() string {
	return strings.TrimSpace(versionRaw)
}
