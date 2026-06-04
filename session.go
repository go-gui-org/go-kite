package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const sessionFile = ".kite.toml"

func loadSession() (BSkySession, error) {
	path := getSessionPath()
	var session BSkySession
	if _, err := toml.DecodeFile(path, &session); err != nil {
		return BSkySession{}, err
	}
	return session, nil
}

func saveSession(session BSkySession) error {
	path := getSessionPath()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return toml.NewEncoder(f).Encode(session)
}

func getSessionPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return sessionFile
	}
	return filepath.Join(homeDir, sessionFile)
}

func isValidSession(session BSkySession) bool {
	return session.Handle != "" && session.Email != "" &&
		session.AccessJWT != "" && session.RefreshJWT != ""
}

func refreshSession(session BSkySession) (BSkySession, error) {
	refresh, err := refreshBSkySession(session)
	if err != nil {
		return BSkySession{}, err
	}
	updated := session
	updated.AccessJWT = refresh.AccessJWT
	updated.RefreshJWT = refresh.RefreshJWT
	if err := saveSession(updated); err != nil {
		return BSkySession{}, fmt.Errorf("save refreshed session: %w", err)
	}
	return updated, nil
}
