package main

import (
	"runtime/debug"
	"strings"
)

const (
	appName = "Kite"
	repoURL = "https://github.com/go-gui-org/go-kite"
)

// version is the git tag, stamped at link time with
// -ldflags "-X main.version=...". The Makefile and
// packaging/package.sh both stamp it from `git describe --tags`, so
// every make-built binary and release artifact carries its tag. Only
// a bare `go run .` or `go build` skips the stamp; those fall back to
// the module version Go records from git (see resolveVersion).
var version string

// appVersion is resolved once at startup. Build info does not change
// while the process runs, so there is no reason to read it per frame.
var appVersion = func() string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(version, info, ok)
}()

// resolveVersion picks the version the about section shows. The
// link-time stamp wins because it is what the release names. Next is
// the main module version Go embeds from git: a tag on a clean
// checkout, a pseudo-version otherwise. "(devel)" means Go found no
// VCS data, which says nothing to a user, so it becomes "dev". A
// leading "v" is trimmed so both sources print the same shape.
// Parameters, not globals, keep every branch testable.
func resolveVersion(stamped string, info *debug.BuildInfo, ok bool) string {
	v := stamped
	if v == "" && ok && info != nil && info.Main.Version != "(devel)" {
		v = info.Main.Version
	}
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return "dev"
	}
	return v
}
