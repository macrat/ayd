package meta

var (
	// Version is the semantic version of the application.
	// This value is injected at build time via ldflags.
	Version = "HEAD"

	// Commit is the git commit hash.
	// This value is injected at build time via ldflags.
	Commit = "UNKNOWN"
)
