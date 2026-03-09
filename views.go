package main

import (
	"fmt"
	"strings"

	"github.com/mike-ward/go-gui/gui"
)

const (
	fieldWidth       = 250
	timelineScrollID = 1
	lineThickness    = 0.5
	maxTimelinePosts = 25
)

var (
	postTextColor    = gui.RGB(0xB0, 0xB0, 0xB0)
	postDividerColor = gui.RGB(0x70, 0x70, 0x70)
)

func loginView(w *gui.Window) gui.View {
	ww, wh := w.WindowSize()
	app := gui.State[App](w)

	return gui.Column(gui.ContainerCfg{
		Width:   float32(ww),
		Height:  float32(wh),
		Sizing:  gui.FixedFixed,
		HAlign:  gui.HAlignCenter,
		Spacing: gui.Some(float32(gui.PadLarge)),
		Content: []gui.View{
			gui.Text(gui.TextCfg{Text: "Login", TextStyle: gui.CurrentTheme().B1}),
			gui.Input(gui.InputCfg{
				Text:        app.UserName,
				Placeholder: "User Name",
				IDFocus:     1,
				Sizing:      gui.FixedFit,
				Width:       fieldWidth,
				OnTextChanged: func(_ *gui.Layout, s string, w *gui.Window) {
					gui.State[App](w).UserName = s
				},
			}),
			gui.Input(gui.InputCfg{
				IsPassword:  true,
				Text:        app.Password,
				Placeholder: "Password",
				IDFocus:     2,
				Sizing:      gui.FixedFit,
				Width:       fieldWidth,
				OnTextChanged: func(_ *gui.Layout, s string, w *gui.Window) {
					gui.State[App](w).Password = s
				},
			}),
			gui.Button(gui.ButtonCfg{
				Disabled: app.LoginPending || strings.TrimSpace(app.UserName) == "" || strings.TrimSpace(app.Password) == "",
				IDFocus:  3,
				Content: []gui.View{
					gui.Text(gui.TextCfg{Text: "Submit"}),
				},
				OnClick: func(_ *gui.Layout, _ *gui.Event, w *gui.Window) {
					app := gui.State[App](w)
					if app.LoginPending {
						return
					}
					app.LoginPending = true
					app.ErrorMsg = ""
					username := app.UserName
					password := app.Password
					go loginAsync(username, password, w)
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
	ww, wh := w.WindowSize()
	content := timelineContent(w)

	pad := gui.NewPadding(1, gui.PadMedium+gui.PadXSmall, gui.PadSmall, gui.PadSmall)
	return gui.Column(gui.ContainerCfg{
		IDFocus:    1,
		IDScroll:   timelineScrollID,
		ScrollMode: gui.ScrollVerticalOnly,
		Width:      float32(ww),
		Height:     float32(wh),
		Sizing:     gui.FixedFixed,
		Padding:    gui.Some(pad),
		OnAnyClick: func(_ *gui.Layout, e *gui.Event, w *gui.Window) {
			if e.MouseButton == gui.MouseRight {
				w.ScrollVerticalTo(timelineScrollID, 0)
				e.IsHandled = true
			}
		},
		Content: []gui.View{
			gui.Column(gui.ContainerCfg{
				Padding: gui.Some(gui.PaddingNone),
				Sizing:  gui.FillFit,
				Spacing: gui.SomeF(3),
				Content: content,
			}),
		},
	})
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
		if strings.TrimSpace(post.FormattedText) == "" && strings.TrimSpace(post.FormattedQuoteText) == "" {
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
			gui.Rectangle(gui.RectangleCfg{Height: gui.PadXSmall - 1}),
			gui.Text(gui.TextCfg{Text: post.FormattedText, Mode: gui.TextModeWrap, TextStyle: postTextStyle}),
		)

		if post.FormattedQuoteText != "" {
			postContent = append(postContent, gui.Row(gui.ContainerCfg{
				Padding:    gui.Some(gui.Padding{Top: gui.PadMedium, Left: 1, Bottom: gui.PadMedium, Right: gui.PadSmall}),
				Sizing:     gui.FillFit,
				SizeBorder: gui.Some(float32(0)),
				Content: []gui.View{
					gui.Rectangle(gui.RectangleCfg{
						Width:  lineThickness,
						Sizing: gui.FixedFill,
						Color:  postTextColor,
					}),
					gui.Column(gui.ContainerCfg{
						Padding: gui.Some(gui.Padding{Right: gui.PadSmall + gui.PadXSmall}),
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
			postContent = append(postContent, textLink(post.LinkTitle, post.LinkURI, postLinkStyle))
		}

		if post.ImagePath != "" && app.ShowImages {
			width := post.ImageWidth
			height := post.ImageHeight
			if width <= 0 {
				width = imageWidth
			}
			if height <= 0 {
				height = maxImageHeight
			}
			if width > imageWidth {
				height = height * (imageWidth / width)
				width = imageWidth
			}
			if height > maxImageHeight {
				height = maxImageHeight
			}

			postContent = append(postContent, gui.Column(gui.ContainerCfg{
				Sizing:  gui.FillFit,
				Padding: gui.Some(gui.PaddingNone),
				Content: []gui.View{
					gui.Image(gui.ImageCfg{
						Src:    post.ImagePath,
						Width:  width,
						Height: height,
					}),
				},
			}))
		}

		postContent = append(postContent,
			gui.Rectangle(gui.RectangleCfg{Height: gui.PadSmall}),
			gui.Rectangle(gui.RectangleCfg{Height: lineThickness, Sizing: gui.FillFixed, Color: postDividerColor}),
		)

		content = append(content, gui.Column(gui.ContainerCfg{
			Padding: gui.Some(gui.PaddingNone),
			Sizing:  gui.FillFit,
			Spacing: gui.SomeF(1),
			Content: postContent,
		}))
	}
	return content
}

func isSafeURI(uri string) bool {
	lower := strings.ToLower(uri)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func textLink(linkTitle, linkURI string, textStyle gui.TextStyle) gui.View {
	return gui.Column(gui.ContainerCfg{
		Padding:    gui.Some(gui.PaddingNone),
		SizeBorder: gui.Some(float32(0)),
		Sizing:     gui.FillFit,
		OnClick: func(_ *gui.Layout, e *gui.Event, w *gui.Window) {
			e.IsHandled = true
			if !isSafeURI(linkURI) {
				return
			}
			np := w.NativePlatformBackend()
			if np == nil {
				return
			}
			if err := np.OpenURI(linkURI); err != nil {
				logError(err.Error())
			}
		},
		OnHover: func(layout *gui.Layout, e *gui.Event, w *gui.Window) {
			e.IsHandled = true
			if len(layout.Children) > 0 && layout.Children[0].Shape != nil && layout.Children[0].Shape.TC != nil {
				ts := layout.Children[0].Shape.TC.TextStyle
				ts.Color = gui.CornflowerBlue
				layout.Children[0].Shape.TC.TextStyle = ts
			}
			w.SetMouseCursorPointingHand()
		},
		Content: []gui.View{
			gui.Text(gui.TextCfg{
				Text:      fmt.Sprintf("%s", linkTitle),
				Mode:      gui.TextModeWrap,
				TextStyle: textStyle,
			}),
		},
	})
}
