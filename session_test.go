package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestSessionRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	in := BSkySession{
		Handle:     "alice",
		Email:      "a@example.com",
		AccessJWT:  "access",
		RefreshJWT: "refresh",
	}
	if err := saveSession(in); err != nil {
		t.Fatalf("saveSession error: %v", err)
	}
	out, err := loadSession()
	if err != nil {
		t.Fatalf("loadSession error: %v", err)
	}
	if out.Handle != in.Handle || out.Email != in.Email || out.AccessJWT != in.AccessJWT || out.RefreshJWT != in.RefreshJWT {
		t.Fatalf("session mismatch\nin=%+v\nout=%+v", in, out)
	}
}

// Logout keeps the handle and email (the file doubles as the settings
// file, and the handle pre-fills the login view) but must drop both
// tokens, so the next start lands on the login view.
func TestClearSessionTokens(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	in := BSkySession{
		Handle:     "alice",
		Email:      "a@example.com",
		AccessJWT:  "access",
		RefreshJWT: "refresh",
	}
	if err := saveSession(in); err != nil {
		t.Fatalf("saveSession error: %v", err)
	}
	if err := clearSessionTokens(in); err != nil {
		t.Fatalf("clearSessionTokens error: %v", err)
	}
	out, err := loadSession()
	if err != nil {
		t.Fatalf("loadSession error: %v", err)
	}
	if out.Handle != in.Handle || out.Email != in.Email {
		t.Fatalf("identity not kept: %+v", out)
	}
	if out.AccessJWT != "" || out.RefreshJWT != "" {
		t.Fatalf("tokens not cleared: %+v", out)
	}
	if isValidSession(out) {
		t.Fatal("cleared session still valid")
	}
}

// refreshSession must not write the session file itself. The timeline
// loop saves the refreshed tokens on the UI thread, and only while the
// loop is still current; a save from the loop goroutine could land
// after logout cleared the file and silently log the user back in.
func TestRefreshSessionDoesNotSave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accessJwt":"new-access","refreshJwt":"new-refresh"}`))
	}))
	defer server.Close()
	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	httpClient = &http.Client{Transport: rewriteTransport{base: baseURL, rt: http.DefaultTransport}}

	got, err := refreshSession(BSkySession{Handle: "alice", RefreshJWT: "old"})
	if err != nil {
		t.Fatalf("refreshSession error: %v", err)
	}
	if got.AccessJWT != "new-access" || got.RefreshJWT != "new-refresh" || got.Handle != "alice" {
		t.Fatalf("unexpected session: %+v", got)
	}
	if _, err := os.Stat(getSessionPath()); !os.IsNotExist(err) {
		t.Fatalf("refreshSession wrote the session file (stat err: %v)", err)
	}
}
