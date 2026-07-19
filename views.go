package main

import (
	"math"
	"strings"

	"github.com/go-gui-org/go-gui/gui"
)

const (
	fieldWidth        = 250
	timelineScrollID  = "timeline"
	timelineContentID = "timeline-content"
	lineThickness     = 0.5
	maxTimelinePosts  = 25
)

var (
	postTextColor    = gui.RGB(0x90, 0x90, 0x90)
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
				ID:          "login-username",
				Text:        app.UserName,
				Placeholder: "User Name",
				Sizing:      gui.FixedFit,
				Width:       fieldWidth,
				OnTextChanged: func(_ *gui.Layout, s string, w *gui.Window) {
					gui.State[App](w).UserName = s
				},
			}),
			gui.Input(gui.InputCfg{
				ID:          "login-password",
				IsPassword:  true,
				Text:        app.Password,
				Placeholder: "Password",
				Sizing:      gui.FixedFit,
				Width:       fieldWidth,
				OnTextChanged: func(_ *gui.Layout, s string, w *gui.Window) {
					gui.State[App](w).Password = s
				},
			}),
			gui.Button(gui.ButtonCfg{
				Disabled:  app.LoginPending || strings.TrimSpace(app.UserName) == "" || strings.TrimSpace(app.Password) == "",
				ID:        "login-submit",
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
		ID:         timelineScrollID,
		Focusable:  true,
		Scrollable: true,
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
		AmendLayout: revealAmend,
		Content: []gui.View{
			gui.Column(gui.ContainerCfg{
				ID:      timelineContentID,
				Padding: gui.Some(gui.PaddingNone),
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

// revealAmend runs after layout positions are computed, before the
// frame renders. When a refresh just prepended posts (RevealAnchorID
// set), it keeps the old content visually in place for this frame and
// then eases the scroll offset to the top, so new posts glide into
// view instead of appearing suddenly. AmendLayout works on absolute
// coordinates and moving a parent does not move its children, so the
// whole content subtree is shifted manually.
func revealAmend(layout *gui.Layout, w *gui.Window) {
	if layout == nil || layout.Shape == nil {
		return
	}
	app := gui.State[App](w)
	if app.RevealAnchorID == "" {
		return
	}
	anchorID := app.RevealAnchorID
	app.RevealAnchorID = ""

	firstID := firstRenderedPostID(app.Timeline)
	if firstID == "" || firstID == anchorID {
		return
	}
	anchor, okAnchor := layout.FindByID(anchorID)
	content, okContent := layout.FindByID(timelineContentID)
	first, okFirst := layout.FindByID(firstID)
	if !okAnchor || !okContent || !okFirst {
		return // anchor post fell off the timeline; jump as before
	}
	if anchor.Shape == nil || first.Shape == nil || content.Shape == nil {
		return
	}

	// Height of the prepended posts: how far the anchor (previously
	// the first post, at the very top of the content column) now sits
	// below the new first post.
	delta := anchor.Shape.Y - first.Shape.Y
	if delta <= 0 || math.IsNaN(float64(delta)) || math.IsInf(float64(delta), 0) {
		return
	}

	// Current displayed offset (<= 0, 0 = top): content top relative
	// to the viewport top.
	viewTop := layout.Shape.Y + layout.Shape.Padding.Top
	offset := content.Shape.Y - viewTop
	if math.IsNaN(float64(offset)) || math.IsInf(float64(offset), 0) {
		return
	}

	// Anchoring must stay within the scrollable range or the offset
	// written below would be clamped, snapping the view next frame.
	viewH := layout.Shape.Height - layout.Shape.Padding.Top - layout.Shape.Padding.Bottom
	maxOffset := viewH - content.Shape.Height // negative when content overflows
	newOffset := offset - delta
	if math.IsNaN(float64(newOffset)) || math.IsInf(float64(newOffset), 0) {
		return
	}
	if maxOffset >= 0 || newOffset < maxOffset {
		return // content fits the viewport or shrank past the user's position
	}

	// Glue the view to the anchor for this frame: shift the already
	// positioned subtree up, and record the matching offset so the
	// next frames lay out identically.
	shiftSubtreeY(content, -delta)
	w.ScrollVerticalTo(timelineScrollID, newOffset)
	// Reading position is now preserved. New posts sit above the
	// viewport; the user scrolls up to see them when ready.
}

// shiftSubtreeY moves a layout and all its descendants vertically.
func shiftSubtreeY(layout *gui.Layout, dy float32) {
	if layout == nil || layout.Shape == nil ||
		math.IsNaN(float64(dy)) || math.IsInf(float64(dy), 0) {
		return
	}
	layout.Shape.Y += dy
	for i := range layout.Children {
		shiftSubtreeY(&layout.Children[i], dy)
	}
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
			gui.Rectangle(gui.RectangleCfg{Height: gui.PadXSmall - 1}),
			gui.Text(gui.TextCfg{Text: post.FormattedText, Mode: gui.TextModeWrap, TextStyle: postTextStyle}),
		)

		if post.FormattedQuoteText != "" {
			postContent = append(postContent, gui.Row(gui.ContainerCfg{
				Padding:    gui.Some(gui.Padding{Top: gui.PadMedium, Left: 1, Bottom: gui.PadMedium, Right: gui.PadSmall}),
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
			// Stable ID so revealAmend can locate posts across refreshes.
			ID:      postViewID(post),
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
				Text:      linkTitle,
				Mode:      gui.TextModeWrap,
				TextStyle: textStyle,
			}),
		},
	})
}
