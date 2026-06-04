package main

import (
	"os"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-gui/gui/backend"
)

const (
	appDefaultWidth  = 300
	appDefaultHeight = 900
)

func main() {
	gui.SetTheme(gui.ThemeDarkBordered)

	app := &App{ShowImages: true}
	processArgs(app)

	w := gui.NewWindow(gui.WindowCfg{
		State:   app,
		Title:   "Kite",
		Width:   appDefaultWidth,
		Height:  appDefaultHeight,
		OnEvent: appOnEvent,
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
	})

	changeFontSize(-0.5, 4, 30, w)
	backend.Run(w)
}

func processArgs(app *App) {
	if len(os.Args) > 1 && os.Args[1] == "-no-images" {
		app.ShowImages = false
	}
}

func appOnEvent(e *gui.Event, w *gui.Window) {
	if e.Type != gui.EventKeyDown || !e.Modifiers.Has(gui.ModAlt) {
		return
	}
	switch e.KeyCode {
	case gui.KeyUp:
		changeFontSize(0.25, 4, 30, w)
		e.IsHandled = true
	case gui.KeyDown:
		changeFontSize(-0.25, 4, 30, w)
		e.IsHandled = true
	}
}
