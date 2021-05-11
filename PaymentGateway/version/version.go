package version

import (
	"strings"
)

const unknown = "unknown"

var (
	buildDate  = unknown
	commitHash string
	version    string
)

const defaultVersion = "devel"

// Version returns the build version of this binary
func Version() string {
	var v []string

	if version == "" {
		version = defaultVersion
	}
	v = append(v, version)

	if commitHash != "" {
		v = append(v, commitHash)
	}

	return strings.Join(v, "-")
}

// BuildDate returns the date this binary was built
func BuildDate() string {
	return buildDate
}
