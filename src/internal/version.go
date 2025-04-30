package internal

import (
	"fmt"
	"runtime"
)

// go build -ldflags "-X 'version.BuildTime=$(date -u)'"

// Will be set at build time using -ldflags
var (
	Version = "v0.1.0"

	// Will be set at build time using -ldflags
	BuildType = "dev"

	// Time the binary was built (set during build with -ldflags).
	BuildTime string

	GitCommit = "unknown"
)

// VersionInfo returns a formatted string with version information
func VersionInfo() string {
	return fmt.Sprintf(
		"Version: %s\nBuild Date: %s\nGit Commit: %s\nGo Version: %s\nOS/Arch: %s/%s",
		Version,
		BuildTime,
		GitCommit,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}
