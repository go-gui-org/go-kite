package main

import (
	"testing"

	"github.com/go-gui-org/go-gui/gui"
)

func TestHelpShortcut(t *testing.T) {
	superSlash := &gui.Event{KeyCode: gui.KeySlash, Modifiers: gui.ModSuper}
	plainSlash := &gui.Event{KeyCode: gui.KeySlash}
	ctrlSuperSlash := &gui.Event{KeyCode: gui.KeySlash, Modifiers: gui.ModSuper | gui.ModCtrl}
	altSuperSlash := &gui.Event{KeyCode: gui.KeySlash, Modifiers: gui.ModSuper | gui.ModAlt}
	f1 := &gui.Event{KeyCode: gui.KeyF1}
	altF1 := &gui.Event{KeyCode: gui.KeyF1, Modifiers: gui.ModAlt}

	// macOS and Linux: Super+/ opens help, plain / or Ctrl+/ / Alt+/
	// don't, and F1 is inert.
	for _, goos := range []string{"darwin", "linux"} {
		if !helpShortcutFor(goos, superSlash) {
			t.Fatalf("%s: Super+/ not recognized", goos)
		}
		if helpShortcutFor(goos, plainSlash) {
			t.Fatalf("%s: plain / must not open help", goos)
		}
		if helpShortcutFor(goos, ctrlSuperSlash) {
			t.Fatalf("%s: Ctrl+Super+/ must not open help", goos)
		}
		if helpShortcutFor(goos, altSuperSlash) {
			t.Fatalf("%s: Alt+Super+/ must not open help", goos)
		}
		if helpShortcutFor(goos, f1) {
			t.Fatalf("%s: F1 must not open help", goos)
		}
	}

	// Windows: plain F1 opens help; Super+/ is the OS's own shortcut.
	if !helpShortcutFor("windows", f1) {
		t.Fatal("windows: F1 not recognized")
	}
	if helpShortcutFor("windows", superSlash) {
		t.Fatal("windows: Super+/ must not open help")
	}
	if helpShortcutFor("windows", altF1) {
		t.Fatal("windows: Alt+F1 must not open help")
	}
}

// The help view teaches these strings, so a label that drifts from
// its matcher is a documented key that does nothing. Each label is
// checked against the matcher it advertises.
func TestShortcutLabelsMatchMatchers(t *testing.T) {
	events := map[string]*gui.Event{
		"F1":      {KeyCode: gui.KeyF1},
		"Cmd+/":   {KeyCode: gui.KeySlash, Modifiers: gui.ModSuper},
		"Super+/": {KeyCode: gui.KeySlash, Modifiers: gui.ModSuper},
		"Cmd+,":   {KeyCode: gui.KeyComma, Modifiers: gui.ModSuper},
		"Ctrl+,":  {KeyCode: gui.KeyComma, Modifiers: gui.ModCtrl},
	}

	for _, goos := range []string{"darwin", "linux", "windows"} {
		label := helpShortcutLabel(goos)
		e, ok := events[label]
		if !ok {
			t.Fatalf("%s: help label %q has no event to match", goos, label)
		}
		if !helpShortcutFor(goos, e) {
			t.Errorf("%s: help label %q does not match helpShortcutFor", goos, label)
		}

		label = settingsShortcutLabel(goos)
		e, ok = events[label]
		if !ok {
			t.Fatalf("%s: settings label %q has no event to match", goos, label)
		}
		if !settingsShortcutFor(goos, e) {
			t.Errorf("%s: settings label %q does not match settingsShortcutFor", goos, label)
		}
	}
}

func TestSettingsEditorCmd(t *testing.T) {
	// Every supported platform must produce a command whose last
	// argument is the settings path; anything else must return nil so
	// openSettingsFile logs instead of spawning garbage.
	for _, goos := range []string{"darwin", "linux", "windows"} {
		cmd := settingsEditorCmd(goos, "/tmp/kite-settings.toml")
		if cmd == nil {
			t.Fatalf("%s: no editor command", goos)
		}
		if got := cmd.Args[len(cmd.Args)-1]; got != "/tmp/kite-settings.toml" {
			t.Errorf("%s: path not last arg, got %q", goos, got)
		}
	}
	if cmd := settingsEditorCmd("plan9", "/tmp/x.toml"); cmd != nil {
		t.Errorf("plan9: want nil command, got %v", cmd.Args)
	}
}

func TestSettingsShortcut(t *testing.T) {
	superComma := &gui.Event{KeyCode: gui.KeyComma, Modifiers: gui.ModSuper}
	ctrlComma := &gui.Event{KeyCode: gui.KeyComma, Modifiers: gui.ModCtrl}
	plainComma := &gui.Event{KeyCode: gui.KeyComma}
	ctrlSuperComma := &gui.Event{KeyCode: gui.KeyComma, Modifiers: gui.ModCtrl | gui.ModSuper}
	altCtrlComma := &gui.Event{KeyCode: gui.KeyComma, Modifiers: gui.ModCtrl | gui.ModAlt}

	// macOS: Cmd+, opens settings; Ctrl+, is a text-edit combo there.
	if !settingsShortcutFor("darwin", superComma) {
		t.Fatal("darwin: Cmd+, not recognized")
	}
	if settingsShortcutFor("darwin", ctrlComma) {
		t.Fatal("darwin: Ctrl+, must not open settings")
	}
	if settingsShortcutFor("darwin", plainComma) {
		t.Fatal("darwin: plain comma must not open settings")
	}

	// Windows and Linux: Ctrl+, opens settings; Super stays out of it.
	for _, goos := range []string{"windows", "linux"} {
		if !settingsShortcutFor(goos, ctrlComma) {
			t.Fatalf("%s: Ctrl+, not recognized", goos)
		}
		if settingsShortcutFor(goos, superComma) {
			t.Fatalf("%s: Super+, must not open settings", goos)
		}
		if settingsShortcutFor(goos, ctrlSuperComma) {
			t.Fatalf("%s: Ctrl+Super+, must not open settings", goos)
		}
		if settingsShortcutFor(goos, altCtrlComma) {
			t.Fatalf("%s: Ctrl+Alt+, must not open settings", goos)
		}
	}
}
