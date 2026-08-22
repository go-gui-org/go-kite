package main

import (
	"os"
	"time"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-gui/gui/backend"
)

const (
	appDefaultWidth  = 300
	appDefaultHeight = 900
)

func main() {
	app := &App{ShowImages: true}
	processArgs(app)

	w := gui.NewWindow(kiteWindowCfg(app))
	// Startup font tuning (see textutil.go): pins the window's theme to a
	// smaller text size so the 300px-wide timeline fits more content.
	// Cannot live in kiteWindowCfg — it needs the window handle, which
	// only exists after NewWindow. Without it the app opens at the
	// theme's default size and only changes once the user presses Alt+↑.
	changeFontSize(-2.25, 4, 30, w)
	backend.Run(w)
}

// kiteWindowCfg is split out of main so TestWindowCfgWiring can inspect the
// config without opening a window. The icon fields in particular are silent
// when wrong — they produce the wrong artwork, not an error.
func kiteWindowCfg(app *App) gui.WindowCfg {
	return gui.WindowCfg{
		State:   app,
		Title:   "Kite",
		Width:   appDefaultWidth,
		Height:  appDefaultHeight,
		OnEvent: appOnEvent,
		// See icon.go: without IconPNG the macOS backend installs go-gui's
		// default icon over Kite's at runtime, bundle or no bundle.
		IconPNG: appIconPNG,
		// X11 keys the taskbar icon off WM_CLASS matched against a .desktop
		// file's StartupWMClass. Without this the hicolor icons install but
		// the window never picks one up. Must match assets/go-kite.desktop.
		WMClass: "go-kite",
		OnInit: func(w *gui.Window) {
			app := gui.State[App](w)
			session, err := loadSession()
			if err != nil {
				app.ErrorMsg = err.Error()
			}
			if isValidSession(session) {
				app.Session = session
				app.startTimelineLoop(w)
			} else {
				w.UpdateView(loginView)
			}
		},
	}
}

func processArgs(app *App) {
	if len(os.Args) > 1 && os.Args[1] == "-no-images" {
		app.ShowImages = false
	}
}

func appOnEvent(e *gui.Event, w *gui.Window) {
	// Presence tracking for the idle-reveal gate in
	// anchorTimelineReveal. Only
	// unhandled events arrive here, but a user at the machine emits a
	// steady stream of them (mouse moves, key ups); programmatic
	// scrolls and animation ticks emit none.
	app := gui.State[App](w)
	app.LastInteraction = time.Now()

	if e.Type != gui.EventKeyDown || !e.Modifiers.Has(gui.ModAlt) {
		return
	}

	switch e.KeyCode {
	case gui.KeyUp:
		changeFontSize(0.25, 4, 30, w)
	case gui.KeyDown:
		changeFontSize(-0.25, 4, 30, w)
	case gui.KeyI:
		// Rendering follows immediately (views.go reads app.ShowImages
		// each frame); the download pass follows on the next poll —
		// timelineLoop re-reads the flag every iteration via
		// readShowImages, so turning images on starts downloading and
		// turning them off stops it.
		app.ShowImages = !app.ShowImages
	}
}
