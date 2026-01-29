// Package cmd provides the command line interface.
// Build variables are set via ldflags during build time.
package cmd

// These variables are set via ldflags during build.
// Example: go build -ldflags "-X github.com/sters/cc-session-tailing/cmd.version=1.0.0".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)
