package main

import "testing"

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
