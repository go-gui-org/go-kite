package main

import (
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/go-gui-org/go-gui/gui"
)

const (
	fieldWidth        = 250
	timelineScrollID  = "timeline"
	timelineContentID = "timeline-content"
	lineThickness     = 0.5
	maxTimelinePosts  = 25

	// helpScrollID is the help view's scroll container. Right-click
	// jumps it back to the top, like the timeline's.
	helpScrollID = "help-scroll"

	// idleRevealAfter: with no user interaction for this long, a
	// prepend reveals new posts (scrolls to top) even from a held
	// position.
	idleRevealAfter = 10 * time.Minute
)

var (
	postTextColor    = gui.RGB(0x90, 0x90, 0x90)
	postDividerColor = gui.RGB(0x70, 0x70, 0x70)
)

func loginView(w *gui.Window) gui.View {
	app := gui.State[App](w)

	return gui.Column(gui.ContainerCfg{
		Sizing:  gui.FillFill,
		HAlign:  gui.HAlignCenter,
		Spacing: gui.Some(float32(gui.PadLarge)),
		Content: []gui.View{
			gui.Text(gui.TextCfg{Text: "Login", TextStyle: gui.CurrentTheme().B1}),
			gui.Input(gui.InputCfg{
				ID:          "login-username",
				Text:        app.UserName,
				Placeholder: "User Name",
				Sizing:      gui.FixedFit,
				Width:       fieldWidth,
				OnTextChanged: func(s string, ctx gui.EventCtx) {
					gui.State[App](ctx.Window).UserName = s
				},
			}),
			gui.Input(gui.InputCfg{
				ID:          "login-password",
				IsPassword:  true,
				Text:        app.Password,
				Placeholder: "Password",
				Sizing:      gui.FixedFit,
				Width:       fieldWidth,
				OnTextChanged: func(s string, ctx gui.EventCtx) {
					gui.State[App](ctx.Window).Password = s
				},
			}),
			gui.Button(gui.ButtonCfg{
				Disabled: app.LoginPending || strings.TrimSpace(app.UserName) == "" || strings.TrimSpace(app.Password) == "",
				ID:       "login-submit",
				Content: []gui.View{
					gui.Text(gui.TextCfg{Text: "Submit"}),
				},
				OnClick: func(ctx gui.EventCtx) {
					app := gui.State[App](ctx.Window)
					if app.LoginPending {
						return
					}
					app.LoginPending = true
					app.ErrorMsg = ""
					username := app.UserName
					password := app.Password
					go loginAsync(username, password, ctx.Window)
				},
			}),
			gui.Text(gui.TextCfg{
				Text:      app.ErrorMsg,
				TextStyle: gui.CurrentTheme().B3,
				Mode:      gui.TextModeWrap,
			}),
		},
	})
}

func loginAsync(username, password string, w *gui.Window) {
	session, err := createSession(username, password)
	w.QueueCommand(func(w *gui.Window) {
		app := gui.State[App](w)
		app.LoginPending = false
		if err != nil {
			app.ErrorMsg = err.Error()
			w.UpdateWindow()
			return
		}
		if saveErr := saveSession(session); saveErr != nil {
			app.ErrorMsg = saveErr.Error()
			w.UpdateWindow()
			return
		}
		app.UserName = ""
		app.Password = ""
		app.ErrorMsg = ""
		app.Session = session
		app.startTimelineLoop(w)
		w.UpdateWindow()
	})
}

func timelineView(w *gui.Window) gui.View {
	content := timelineContent(w)

	pad := gui.NewPadding(1, gui.PadMedium+gui.PadXSmall, gui.PadSmall, gui.PadSmall)
	return gui.Column(gui.ContainerCfg{
		ID:         timelineScrollID,
		Focusable:  true,
		Scrollable: true,
		ScrollMode: gui.ScrollVerticalOnly,
		Sizing:     gui.FillFill,
		Padding:    pad,
		OnAnyClick: func(ctx gui.EventCtx) {
			if ctx.Event.MouseButton == gui.MouseRight {
				ctx.Window.ScrollVerticalTo(timelineScrollID, 0)
				ctx.Consume()
			}
		},
		Content: []gui.View{
			gui.Column(gui.ContainerCfg{
				ID:      timelineContentID,
				Padding: gui.PaddingNone,
				Sizing:  gui.FillFit,
				Spacing: gui.SomeF(3),
				Content: content,
			}),
		},
	})
}

// postViewID returns a stable, unique view ID for a post. The URI
// alone is not unique: a timeline may hold a post and a repost of it
// (or reposts by different users), so the reposter disambiguates.
func postViewID(post Post) string {
	return post.ID + "\x00" + post.RepostBy
}

// postIsRendered mirrors timelineContent's skip rule so anchor math
// operates on the same posts the view actually shows.
func postIsRendered(post Post) bool {
	return strings.TrimSpace(post.FormattedText) != "" ||
		strings.TrimSpace(post.FormattedQuoteText) != ""
}

// firstRenderedPostID returns the view ID of the first post the
// timeline renders, or "" when nothing renders.
func firstRenderedPostID(t Timeline) string {
	for _, post := range t.Posts {
		if postIsRendered(post) {
			return postViewID(post)
		}
	}
	return ""
}

func timelineContent(w *gui.Window) []gui.View {
	app := gui.State[App](w)
	content := make([]gui.View, 0, maxTimelinePosts)

	if len(app.Timeline.Posts) == 0 {
		content = append(content, gui.Column(gui.ContainerCfg{
			Sizing: gui.FillFill,
			HAlign: gui.HAlignCenter,
			VAlign: gui.VAlignMiddle,
			Content: []gui.View{
				gui.Text(gui.TextCfg{Text: "Fetching Timeline..."}),
			},
		}))
		return content
	}

	baseTextStyle := gui.CurrentTheme().N3
	postTextStyle := baseTextStyle
	postTextStyle.Color = postTextColor
	postLinkStyle := baseTextStyle
	postLinkStyle.Color = gui.CornflowerBlue
	postLinkStyle.Size = baseTextStyle.Size - 1
	postRepostStyle := baseTextStyle
	postRepostStyle.Color = postTextColor
	postRepostStyle.Size = baseTextStyle.Size - 1

	for _, post := range app.Timeline.Posts {
		if !postIsRendered(post) {
			continue
		}

		postContent := make([]gui.View, 0, 10)
		if post.FormattedRepostBy != "" {
			postContent = append(postContent, gui.Text(gui.TextCfg{
				Text:      post.FormattedRepostBy,
				Mode:      gui.TextModeWrap,
				TextStyle: postRepostStyle,
			}))
		}

		postContent = append(postContent,
			textLink(post.FormattedTimeAuthor, post.BSkyLinkURI, baseTextStyle),
			gui.Rectangle(gui.RectangleCfg{Height: 2, Width: 1}), // spacer
			gui.Text(gui.TextCfg{Text: post.FormattedText, Mode: gui.TextModeWrap, TextStyle: postTextStyle}),
		)

		if post.FormattedQuoteText != "" {
			postContent = append(postContent, gui.Row(gui.ContainerCfg{
				Padding:    gui.NewPadding(gui.PadMedium, gui.PadSmall, gui.PadMedium, 1),
				Sizing:     gui.FillFit,
				Spacing:    gui.SomeF(7.5),
				SizeBorder: gui.Some(float32(0)),
				Content: []gui.View{
					gui.Rectangle(gui.RectangleCfg{
						Width:  lineThickness,
						Sizing: gui.FixedFill,
						Color:  postTextColor,
					}),
					gui.Column(gui.ContainerCfg{
						Padding: gui.NewPadding(0, gui.PadSmall+gui.PadXSmall, 0, 0),
						Sizing:  gui.FillFit,
						Spacing: gui.Some(float32(0)),
						Content: []gui.View{
							textLink(post.FormattedQuoteTimeAuth, post.QuotePostLinkURI, baseTextStyle),
							gui.Rectangle(gui.RectangleCfg{Height: gui.PadXSmall - 1}),
							gui.Text(gui.TextCfg{Text: post.FormattedQuoteText, Mode: gui.TextModeWrap, TextStyle: postTextStyle}),
						},
					}),
				},
			}))
		}

		if post.LinkURI != "" {
			postContent = append(postContent,
				gui.Rectangle(gui.RectangleCfg{Height: 2, Width: 1}), // spacer
				textLink(post.LinkTitle, post.LinkURI, postLinkStyle))
		}

		if post.ImagePath != "" && app.ShowImages {
			width, height := scaledImageDims(post.ImageWidth, post.ImageHeight)

			postContent = append(postContent, gui.Column(gui.ContainerCfg{
				Sizing:  gui.FillFit,
				Padding: gui.PaddingNone,
				Spacing: gui.NoSpacing,
				Content: []gui.View{
					gui.Image(gui.ImageCfg{
						Src:    post.ImagePath,
						Width:  width,
						Height: height,
					}),
					gui.Rectangle(gui.RectangleCfg{Height: 3}),
				},
			}))
		}

		postContent = append(postContent,
			gui.Rectangle(gui.RectangleCfg{Height: 1}),
			gui.Rectangle(gui.RectangleCfg{Height: lineThickness, Sizing: gui.FillFixed, Color: postDividerColor}),
		)

		content = append(content, gui.Column(gui.ContainerCfg{
			// Stable ID so revealAmend can locate posts across refreshes.
			ID:      postViewID(post),
			Padding: gui.PaddingNone,
			Sizing:  gui.FillFit,
			Spacing: gui.SomeF(1),
			Content: postContent,
		}))
	}
	return content
}

// scaledImageDims fits a post embed's intrinsic dimensions to the
// timeline's display box: wide images scale down proportionally,
// missing (zero/negative) dims get display defaults, height is capped,
// and a 1px floor keeps degenerate (absurdly wide) embeds visible
// instead of vanishing. Defaults apply after scaling so a missing
// height yields the full display height, not a proportionally shrunk one.
//
// NaN is folded to zero up front: every comparison against NaN is
// false, so without this a NaN dim would sail through all four branches
// and poison the layout with NaN extents. (The current data path can't
// produce NaN — imageDimensions decodes from disk — but the function
// must be safe for any float.)
func scaledImageDims(width, height float32) (float32, float32) {
	if math.IsNaN(float64(width)) {
		width = 0
	}
	if math.IsNaN(float64(height)) {
		height = 0
	}
	if width <= 0 {
		width = imageWidth
	}
	if width > imageWidth {
		height = height * (imageWidth / width)
		width = imageWidth
	}
	if height <= 0 {
		height = maxImageHeight
	}
	if height > maxImageHeight {
		height = maxImageHeight
	}
	if height < 1 {
		height = 1
	}
	return width, height
}

func isSafeURI(uri string) bool {
	lower := strings.ToLower(uri)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// helpItem is one row of the help view: a shortcut (or mouse action)
// and a plain-language description of what it does. View-internal
// only — models.go is reserved for atproto lexicon mirrors.
type helpItem struct {
	key   string
	label string
}

// helpShortcutLabel and settingsShortcutLabel print the shortcuts the
// help view lists. They mirror helpShortcutFor and
// settingsShortcutFor — change a matcher and change its label with
// it, or the help view teaches a key that does nothing. The goos
// parameter keeps both testable on any host.
func helpShortcutLabel(goos string) string {
	switch goos {
	case "windows":
		return "F1"
	case "darwin":
		return "Cmd+/"
	default:
		return "Super+/"
	}
}

func settingsShortcutLabel(goos string) string {
	if goos == "darwin" {
		return "Cmd+,"
	}
	return "Ctrl+,"
}

// helpShortcutFor reports whether the event is the help shortcut for
// the given OS. macOS and Linux use Super+/ (Cmd+/); Windows uses
// plain F1, because the Super key there belongs to the OS (Win+/ is a
// system shortcut). Modifiers are excluded on both paths so combos
// that mean something else — Ctrl+/ (find), Alt+/ (menu mnemonic),
// Alt+F1 — don't get swallowed as a help toggle. The goos parameter
// keeps the mapping testable on any host.
func helpShortcutFor(goos string, e *gui.Event) bool {
	if goos == "windows" {
		return e.KeyCode == gui.KeyF1 &&
			!e.Modifiers.HasAny(gui.ModSuper, gui.ModCtrl, gui.ModAlt)
	}
	return e.KeyCode == gui.KeySlash &&
		e.Modifiers.Has(gui.ModSuper) &&
		!e.Modifiers.HasAny(gui.ModCtrl, gui.ModAlt)
}

func helpShortcutPressed(e *gui.Event) bool {
	return helpShortcutFor(runtime.GOOS, e)
}

// toggleHelp swaps the current view for the help view and back.
// Opening stores nothing — app.CurrentView already holds the pre-help
// view. The timeline loop normally only refreshes with UpdateWindow,
// but its give-up path does install loginView over the help view; it
// clears ShowHelp when it does, so the next toggle opens rather than
// closing an already-gone help view. Closing restores CurrentView,
// falling back to loginView if nothing was recorded.
func toggleHelp(w *gui.Window) {
	app := gui.State[App](w)
	if app.ShowHelp {
		app.ShowHelp = false
		restore := app.CurrentView
		if restore == nil {
			restore = loginView
		}
		w.UpdateView(restore)
		return
	}
	app.ShowHelp = true
	w.UpdateView(helpView)
}

// helpView lists the app's shortcuts and mouse gestures. The window
// is only ~300px wide, so every description is wrapped text and no
// row carries a fixed width; the whole view scrolls, and right-click
// jumps back to the top just like the timeline.
func helpView(w *gui.Window) gui.View {
	theme := gui.CurrentTheme()

	helpKey := helpShortcutLabel(runtime.GOOS)
	settingsKey := settingsShortcutLabel(runtime.GOOS)

	content := []gui.View{
		gui.Text(gui.TextCfg{Text: "Help", TextStyle: theme.B1}),
		gui.Rectangle(gui.RectangleCfg{Height: gui.PadSmall}),
		helpSection(theme, "Keyboard", []helpItem{
			{key: helpKey, label: "Open or close this help view"},
			{key: settingsKey, label: "Open the settings file in an editor"},
			{key: "Alt+Up", label: "Increase font size"},
			{key: "Alt+Down", label: "Decrease font size"},
			{key: "Alt+I", label: "Toggle image loading"},
		}),
		gui.Rectangle(gui.RectangleCfg{Height: gui.PadMedium}),
		helpSection(theme, "Mouse", []helpItem{
			{key: "Right-click", label: "Scroll back to the top of the timeline"},
			{key: "Left-click", label: "Open links in your browser"},
		}),
	}

	return gui.Column(gui.ContainerCfg{
		ID:         helpScrollID,
		Focusable:  true,
		Scrollable: true,
		ScrollMode: gui.ScrollVerticalOnly,
		Sizing:     gui.FillFill,
		Padding:    gui.NewPadding(gui.PadSmall, gui.PadMedium, gui.PadSmall, gui.PadMedium),
		OnAnyClick: func(ctx gui.EventCtx) {
			if ctx.Event.MouseButton == gui.MouseRight {
				ctx.Window.ScrollVerticalTo(helpScrollID, 0)
				ctx.Consume()
			}
		},
		Content: []gui.View{
			gui.Column(gui.ContainerCfg{
				Padding: gui.PaddingNone,
				Sizing:  gui.FillFit,
				Spacing: gui.SomeF(2),
				Content: content,
			}),
		},
	})
}

// helpSection renders one titled block of shortcut rows: a divider
// under the title, then key/description pairs. Keys are bold,
// descriptions muted and wrapped, so rows reflow inside a 300px-wide
// window instead of spilling off the right edge. Spacers between
// blocks keep the pairs visually grouped.
func helpSection(theme gui.Theme, title string, items []helpItem) gui.View {
	keyStyle := theme.B3
	descStyle := theme.N3
	descStyle.Color = postTextColor

	children := make([]gui.View, 0, 1+len(items)*3)
	children = append(children,
		gui.Text(gui.TextCfg{Text: title, TextStyle: theme.B3}),
		gui.Rectangle(gui.RectangleCfg{Height: lineThickness, Sizing: gui.FillFixed, Color: postDividerColor}),
	)
	for _, item := range items {
		children = append(children,
			gui.Rectangle(gui.RectangleCfg{Height: gui.PadSmall}),
			gui.Text(gui.TextCfg{Text: item.key, TextStyle: keyStyle}),
			gui.Text(gui.TextCfg{Text: item.label, TextStyle: descStyle, Mode: gui.TextModeWrap}),
		)
	}
	return gui.Column(gui.ContainerCfg{
		Padding: gui.PaddingNone,
		Sizing:  gui.FillFit,
		Spacing: gui.SomeF(1),
		Content: children,
	})
}

func textLink(linkTitle, linkURI string, textStyle gui.TextStyle) gui.View {
	return gui.Column(gui.ContainerCfg{
		Padding:    gui.PaddingNone,
		SizeBorder: gui.Some(float32(0)),
		Sizing:     gui.FillFit,
		OnClick: func(ctx gui.EventCtx) {
			ctx.Consume()
			if ctx.Event.MouseButton != gui.MouseLeft {
				return
			}
			if !isSafeURI(linkURI) {
				return
			}
			np := ctx.Window.NativePlatformBackend()
			if np == nil {
				return
			}
			if err := np.OpenURI(linkURI); err != nil {
				logError(err.Error())
			}
		},
		OnHover: func(ctx gui.EventCtx) {
			ctx.Consume()
			if len(ctx.Layout.Children) > 0 && ctx.Layout.Children[0].Shape != nil && ctx.Layout.Children[0].Shape.TC != nil {
				ts := ctx.Layout.Children[0].Shape.TC.TextStyle
				ts.Color = gui.CornflowerBlue
				ctx.Layout.Children[0].Shape.TC.TextStyle = ts
			}
			ctx.Window.SetMouseCursorPointingHand()
		},
		Content: []gui.View{
			gui.Text(gui.TextCfg{
				Text:      linkTitle,
				Mode:      gui.TextModeWrap,
				TextStyle: textStyle,
			}),
		},
	})
}
