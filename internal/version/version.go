// Package version provides build-time version information
// This package is used by the build system to embed version details
package version

var (
	// Version is the semantic version, set by build flags
	Version = "v0.1.0-dev"

	// BuildTime is when the binary was built, set by build flags
	BuildTime = "unknown"

	// Commit is the git commit hash, set by build flags
	Commit = "unknown"
)

// Info returns formatted version information
func Info() string {
	return "AI Work Studio " + Version + " (built " + BuildTime + ", commit " + Commit + ")"
}

// Short returns just the version string
func Short() string {
	return Version
}