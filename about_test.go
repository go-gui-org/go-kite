package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	tagged := &debug.BuildInfo{Main: debug.Module{Version: "v0.2.1"}}
	devel := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}
	empty := &debug.BuildInfo{}

	tests := []struct {
		name    string
		stamped string
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{"stamped wins over build info", "1.2.0", tagged, true, "1.2.0"},
		{"stamped leading v trimmed", "v1.2.0", nil, false, "1.2.0"},
		{"build info tag", "", tagged, true, "0.2.1"},
		{"devel falls back", "", devel, true, "dev"},
		{"empty build info falls back", "", empty, true, "dev"},
		{"no build info falls back", "", nil, false, "dev"},
		{"nil info with ok falls back", "", nil, true, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.stamped, tt.info, tt.ok); got != tt.want {
				t.Fatalf("resolveVersion(%q) = %q, want %q", tt.stamped, got, tt.want)
			}
		})
	}
}

// The help view hands repoURL to textLink, which drops anything
// isSafeURI rejects. A URL that fails the guard is a dead link.
func TestRepoURLPassesSafeURIGuard(t *testing.T) {
	if !isSafeURI(repoURL) {
		t.Fatalf("repoURL %q rejected by isSafeURI", repoURL)
	}
}
