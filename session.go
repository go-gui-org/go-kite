package main

import (
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

// clearSessionTokens rewrites the session file with only the handle
// and email. It rewrites rather than deletes: the same file is the
// settings file openSettingsFile opens, and the handle pre-fills the
// login view. Empty tokens fail isValidSession, so the next start
// shows the login view.
func clearSessionTokens(session BSkySession) error {
	return saveSession(BSkySession{Handle: session.Handle, Email: session.Email})
}

// logout ends the session and returns to the login view. It runs on
// the UI thread (the help view's Log out button), so it changes app
// state directly.
//
// Order matters: the timeline loop is cancelled first, so it cannot
// refresh the tokens and write them back to disk after they are
// cleared. The server revoke runs in a goroutine because it is network
// I/O. It is best-effort: a failure is logged, and the local logout
// stands either way, since the user asked to be logged out of this
// app. The goroutine touches no app state, so it needs no QueueCommand.
func logout(w *gui.Window) {
	app := gui.State[App](w)
	if app.LoopCancel != nil {
		close(app.LoopCancel)
		app.LoopCancel = nil
	}

	old := app.Session
	app.Session = BSkySession{}
	app.Timeline = Timeline{}
	app.Password = ""
	app.ErrorMsg = ""
	app.LoginPending = false
	app.UserName = old.Handle

	if err := clearSessionTokens(old); err != nil {
		// Shown on the login view: the tokens are still on disk, so
		// the next start would log straight back in.
		app.ErrorMsg = "logout: " + err.Error()
	}
	go func() {
		if err := deleteSession(old); err != nil {
			logError("revoke session: " + err.Error())
		}
	}()

	// Logout runs from the help view, so help is closing too — clear
	// ShowHelp so the next toggle opens help instead of closing it.
	app.ShowHelp = false
	app.CurrentView = loginView
	w.SetView(loginView)
}

// refreshSession trades the refresh token for new tokens. It does not
// save them: it runs on the timeline loop goroutine, and a save from
// there could land after logout cleared the file and log the user
// back in on the next start. timelineLoop saves on the UI thread
// instead, and only while its loop is still the current one.
func refreshSession(session BSkySession) (BSkySession, error) {
	refresh, err := refreshBSkySession(session)
	if err != nil {
		return BSkySession{}, err
	}
	updated := session
	updated.AccessJWT = refresh.AccessJWT
	updated.RefreshJWT = refresh.RefreshJWT
	return updated, nil
}
