package cli

import (
	"fmt"
	"strings"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func VersionString() string {
	if version == "" {
		return "dev"
	}
	return version
}

func VersionOutput() string {
	lines := []string{fmt.Sprintf("linear version %s", VersionString())}
	if commit != "" {
		lines = append(lines, fmt.Sprintf("commit %s", commit))
	}
	if date != "" {
		lines = append(lines, fmt.Sprintf("built %s", date))
	}
	return strings.Join(lines, "\n")
}
