package version

import "strings"

// Version is the CLI release. Override at build time with
// -ldflags "-X github.com/buding00/springhere/internal/version.Version=..."
var Version = "0.1.0"

// Display is Version without a leading "v".
func Display() string {
	return strings.TrimPrefix(strings.TrimSpace(Version), "v")
}

// Tag is Version as a git tag, for example v0.1.0.
func Tag() string {
	d := Display()
	if d == "" {
		return ""
	}
	return "v" + d
}
