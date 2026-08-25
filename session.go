package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/go-gui-org/go-gui/gui"
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

// settingsShortcutFor reports whether the event is the settings
// shortcut for the given OS: Cmd+, on macOS, Ctrl+, on Windows and
// Linux — the standard preferences shortcut on each. The other
// modifier is excluded on both so the two never collide (Ctrl+, is a
// text-edit combo on macOS, and Super+, is OS territory on Windows).
// The goos parameter keeps the mapping testable on any host.
func settingsShortcutFor(goos string, e *gui.Event) bool {
	if goos == "darwin" {
		return e.KeyCode == gui.KeyComma &&
			e.Modifiers.Has(gui.ModSuper) &&
			!e.Modifiers.HasAny(gui.ModCtrl, gui.ModAlt)
	}
	return e.KeyCode == gui.KeyComma &&
		e.Modifiers.Has(gui.ModCtrl) &&
		!e.Modifiers.HasAny(gui.ModSuper, gui.ModAlt)
}

func settingsShortcutPressed(e *gui.Event) bool {
	return settingsShortcutFor(runtime.GOOS, e)
}

// settingsEditorCmd returns the command that opens path in the given
// OS's editor: TextEdit on macOS (open -e), the default handler on
// Linux (xdg-open), Notepad on Windows (a .toml has no association
// there). nil means the platform is unsupported. Split out from
// openSettingsFile so the mapping is testable without spawning an
// editor.
func settingsEditorCmd(goos, path string) *exec.Cmd {
	switch goos {
	case "darwin":
		return exec.Command("open", "-e", path)
	case "linux":
		return exec.Command("xdg-open", path)
	case "windows":
		return exec.Command("notepad", path)
	default:
		return nil
	}
}

// openSettingsFile launches the platform editor on the settings file.
// The file is created empty if missing, so the shortcut works before
// the first login; 0o600 because the same file holds session tokens.
// The path comes from getSessionPath — os.UserHomeDir plus a fixed
// name, never user input — so passing it to the platform openers is
// safe.
//
// Start, not Run: this runs on the UI thread, and Notepad (and any
// xdg-open handler that does not fork) only exits when the user
// closes the editor — Run would freeze the window until then. The
// reaping goroutine keeps the finished child from lingering as a
// zombie.
func openSettingsFile() {
	path := getSessionPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			logError("create settings file: " + err.Error())
			return
		}
	}
	cmd := settingsEditorCmd(runtime.GOOS, path)
	if cmd == nil {
		logError("open settings file: unsupported platform " + runtime.GOOS)
		return
	}
	if err := cmd.Start(); err != nil {
		logError("open settings file: " + err.Error())
		return
	}
	go func() { _ = cmd.Wait() }()
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
