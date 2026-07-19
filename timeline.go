package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-gui-org/go-gui/gui"
	"golang.org/x/image/draw"
)

const (
	kiteDir          = "kite"
	maxRetryAttempts = 10
	imageWidth       = 270
	maxImageHeight   = 250
	jpegQuality      = 90
)

var imageTmpDir = filepath.Join(os.TempDir(), kiteDir)
var imageWriteLocks sync.Map

func (app *App) startTimelineLoop(w *gui.Window) {
	if app.LoopCancel != nil {
		close(app.LoopCancel)
	}
	app.LoopCancel = make(chan struct{})
	cancel := app.LoopCancel
	session := app.Session
	showImages := app.ShowImages

	w.UpdateView(timelineView)
	go timelineLoop(w, cancel, session, showImages)
}

func timelineLoop(w *gui.Window, cancel <-chan struct{}, session BSkySession, showImages bool) {
	fallbackCounter := 0

	for {
		select {
		case <-cancel:
			return
		default:
		}

		blueskyTimeline, err := getTimeline(session)
		if err != nil {
			if fallbackCounter < maxRetryAttempts {
				fallbackCounter++
				refreshed, refreshErr := refreshSession(session)
				if refreshErr == nil {
					session = refreshed
					w.QueueCommand(func(w *gui.Window) {
						gui.State[App](w).Session = refreshed
						w.UpdateWindow()
					})
				}
				sleepOrCancel(cancel, time.Duration(fallbackCounter*fallbackCounter)*time.Second)
				continue
			}

			w.QueueCommand(func(w *gui.Window) {
				app := gui.State[App](w)
				app.Timeline = Timeline{}
				app.ErrorMsg = err.Error()
				app.Password = ""
				app.LoginPending = false
				w.UpdateView(loginView)
				w.UpdateWindow()
			})
			return
		}

		pruneDiskImageCache()

		timeline := fromBlueskyTimeline(blueskyTimeline, maxTimelinePosts)
		w.QueueCommand(func(w *gui.Window) {
			app := gui.State[App](w)
			setRevealAnchor(app, timeline)
			app.Timeline = timeline
			app.ErrorMsg = ""
			w.UpdateWindow()
		})

		if showImages {
			getTimelineImages(blueskyTimeline)
			timelineWithImages := fromBlueskyTimeline(blueskyTimeline, maxTimelinePosts)
			w.QueueCommand(func(w *gui.Window) {
				gui.State[App](w).Timeline = timelineWithImages
				w.UpdateWindow()
			})
		}

		fallbackCounter = 0
		sleepOrCancel(cancel, time.Minute)
	}
}

// setRevealAnchor records the post currently at the top of the
// timeline when an incoming refresh moves it down (new posts were
// prepended). revealAmend consumes the anchor on the next layout
// pass to scroll the new posts in instead of jumping. Initial load
// (empty old timeline) and no-change refreshes set nothing.
func setRevealAnchor(app *App, incoming Timeline) {
	oldID := firstRenderedPostID(app.Timeline)
	newID := firstRenderedPostID(incoming)
	if oldID != "" && newID != "" && oldID != newID {
		app.RevealAnchorID = oldID
	}
}

func sleepOrCancel(cancel <-chan struct{}, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-cancel:
		return
	case <-t.C:
		return
	}
}

func fromBlueskyTimeline(timeline bSkyTimeline, maxPosts int) Timeline {
	posts := make([]Post, 0, maxPosts)
	for _, post := range timeline.Posts {
		if post.Post.Record.Reply.Parent.CID != "" || post.Post.Record.Reply.Root.CID != "" {
			continue
		}
		posts = append(posts, fromBlueskyPost(post))
		if len(posts) >= maxPosts {
			break
		}
	}
	return Timeline{Posts: posts}
}

func fromBlueskyPost(post bSkyPost) Post {
	handle := post.Post.Author.Handle
	dName := post.Post.Author.DisplayName
	name := handle
	if dName != "" {
		name = dName
	}

	path, alt, imgW, imgH := postImage(post)
	bskyLinkURI := blueskyPostLink(post)
	repostByName := repostBy(post)

	text := post.Post.Record.Text
	uri, title := externalLink(post)
	inlineURI, byteStart, byteEnd := inlineLink(post)
	if indexesInString(text, byteStart, byteEnd) {
		uri = inlineURI
		title = sanitizeText(text[byteStart:byteEnd])
		text = text[:byteStart] + text[byteEnd:]
	}

	qText := getQuotePostText(post)
	qURI, _, qByteStart, qByteEnd := getQuotePostLink(post)
	if indexesInString(qText, qByteStart, qByteEnd) {
		uri = qURI
		qText = qText[:qByteStart] + qText[qByteEnd:]
	}

	createdAt, err := time.Parse(time.RFC3339, post.Post.Record.CreatedAt)
	if err != nil {
		createdAt = time.Now().UTC()
	}

	quoteCreatedAt := getQuotePostCreatedAt(post)

	formattedRepostBy := ""
	if repostByName != "" {
		formattedRepostBy = truncateLongFields("• reposted by " + repostByName)
	}

	return Post{
		ID:                     post.Post.URI,
		Author:                 name,
		Verified:               blueskyPostVerified(post),
		CreatedAt:              createdAt,
		Text:                   text,
		LinkURI:                uri,
		LinkTitle:              title,
		ImagePath:              path,
		ImageAlt:               alt,
		ImageWidth:             imgW,
		ImageHeight:            imgH,
		RepostBy:               repostByName,
		Replies:                post.Post.Replies,
		Reposts:                post.Post.Reposts + post.Post.Quotes,
		Likes:                  post.Post.Likes,
		BSkyLinkURI:            bskyLinkURI,
		QuotePostAuthor:        getQuotePostAuthor(post),
		QuotePostCreatedAt:     quoteCreatedAt,
		QuotePostText:          qText,
		QuotePostLinkURI:       qURI,
		FormattedText:          sanitizeText(text),
		FormattedRepostBy:      formattedRepostBy,
		FormattedTimeAuthor:    authorTimestampText(name, createdAt),
		FormattedQuoteText:     sanitizeText(qText),
		FormattedQuoteTimeAuth: authorTimestampText(getQuotePostAuthor(post), quoteCreatedAt),
	}
}

func authorTimestampText(author string, createdAt time.Time) string {
	timeShort := relativeShort(createdAt.Local(), time.Now().Local())
	return truncateLongFields(fmt.Sprintf("%s • %s", sanitizeText(author), timeShort))
}

func blueskyPostVerified(post bSkyPost) bool {
	return post.Post.Author.Verification.VerifiedStatus == "valid"
}

func blueskyPostLink(post bSkyPost) string {
	id := post.Post.URI
	if idx := strings.LastIndex(id, ".post/"); idx >= 0 {
		id = id[idx+6:]
	}
	handle := post.Post.Author.Handle
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, id)
}

func repostBy(post bSkyPost) string {
	if strings.Contains(post.Reason.Type, "Repost") {
		if post.Reason.By.DisplayName != "" {
			return post.Reason.By.DisplayName
		}
		return post.Reason.By.Handle
	}
	return ""
}

func externalLink(post bSkyPost) (string, string) {
	external := post.Post.Record.Embed.External
	if external.URI != "" {
		title := external.Title
		if title == "" {
			title = external.URI
		}
		return external.URI, sanitizeText(title)
	}
	return "", ""
}

func postImage(post bSkyPost) (string, string, float32, float32) {
	sources := extractImageSources(post)
	for _, source := range sources {
		tmpFile := imageTmpFilePath(source.CID)
		if tmpFile == "" {
			continue
		}
		if _, err := os.Stat(tmpFile); err == nil {
			w, h := imageDimensions(tmpFile)
			return tmpFile, source.Alt, w, h
		}
	}
	return "", "", 0, 0
}

func imageDimensions(path string) (float32, float32) {
	f, err := os.Open(path)
	if err != nil {
		return imageWidth, maxImageHeight
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil || cfg.Width == 0 || cfg.Height == 0 {
		return imageWidth, maxImageHeight
	}
	return float32(cfg.Width), float32(cfg.Height)
}

func getTimelineImages(timeline bSkyTimeline) {
	if err := os.MkdirAll(imageTmpDir, 0o755); err != nil {
		logError(err.Error())
		return
	}

	var wg sync.WaitGroup
	for _, post := range timeline.Posts {
		p := post
		wg.Add(1)
		go func() {
			defer wg.Done()
			downloadPostImages(p)
		}()
	}
	wg.Wait()
}

func downloadPostImages(post bSkyPost) {
	imageSources := extractImageSources(post)
	for _, source := range imageSources {
		imageTmpFile := imageTmpFilePath(source.CID)
		if imageTmpFile == "" {
			continue
		}
		if _, err := os.Stat(imageTmpFile); err == nil {
			continue
		}

		var blob []byte
		var err error
		if source.URL != "" {
			resp, httpErr := httpClient.Get(source.URL)
			if httpErr != nil {
				logError(httpErr.Error())
				continue
			}
			if resp.StatusCode != http.StatusOK {
				_ = resp.Body.Close()
				continue
			}
			blob, err = ioReadAllClose(resp.Body)
		} else {
			blob, err = getBlob(source.AuthorDID, source.CID)
		}
		if err != nil {
			continue
		}

		lock := imageWriteLock(imageTmpFile)
		lock.Lock()
		if _, err := os.Stat(imageTmpFile); err == nil {
			lock.Unlock()
			continue
		}
		if err := saveImage(imageTmpFile, blob); err != nil {
			lock.Unlock()
			continue
		}
		lock.Unlock()
	}
}

func imageWriteLock(path string) *sync.Mutex {
	lock, _ := imageWriteLocks.LoadOrStore(path, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func ioReadAllClose(r io.ReadCloser) ([]byte, error) {
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

func extractImageSources(post bSkyPost) []imageSource {
	sources := make([]imageSource, 0, 4)
	if len(post.Post.Record.Embed.Images) > 0 {
		for _, img := range post.Post.Record.Embed.Images {
			if img.Image.Ref.Link != "" {
				sources = append(sources, imageSource{
					CID:       img.Image.Ref.Link,
					Alt:       img.Alt,
					AuthorDID: post.Post.Author.DID,
				})
			}
		}
	} else if len(post.Post.Record.Embed.Media.Images) > 0 {
		for _, img := range post.Post.Record.Embed.Media.Images {
			if img.Image.Ref.Link != "" {
				sources = append(sources, imageSource{
					CID:       img.Image.Ref.Link,
					Alt:       img.Alt,
					AuthorDID: post.Post.Author.DID,
				})
			}
		}
	} else if post.Post.Embed.Thumbnail != "" {
		sources = append(sources, imageSource{
			CID:       post.Post.Embed.CID,
			URL:       post.Post.Embed.Thumbnail,
			AuthorDID: post.Post.Author.DID,
		})
	} else if len(post.Post.Embed.Record.Value.Embed.Images) > 0 {
		for _, img := range post.Post.Embed.Record.Value.Embed.Images {
			if img.Image.Ref.Link != "" {
				sources = append(sources, imageSource{
					CID:       img.Image.Ref.Link,
					Alt:       img.Alt,
					AuthorDID: post.Post.Embed.Record.Author.DID,
				})
			}
		}
	}
	return sources
}

func saveImage(name string, blob []byte) error {
	src, _, err := image.Decode(bytes.NewReader(blob))
	if err != nil {
		return err
	}
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return fmt.Errorf("invalid image dimensions")
	}
	srcW := b.Dx()
	srcH := b.Dy()
	ratio := float64(srcH) / float64(srcW)
	targetW := srcW
	if targetW > imageWidth {
		targetW = imageWidth
	}
	targetH := int(math.Round(float64(targetW) * ratio))
	if targetH <= 0 {
		targetH = 1
	}

	scaled := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, src.Bounds(), draw.Over, nil)

	var dst image.Image = scaled
	if targetH > maxImageHeight {
		// Keep full width and clip vertical overflow from the bottom.
		dst = scaled.SubImage(image.Rect(0, 0, targetW, maxImageHeight))
	}

	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return jpeg.Encode(f, dst, &jpeg.Options{Quality: jpegQuality})
}

func hasEmbedPost(post bSkyPost) bool {
	return strings.Contains(post.Post.Embed.Record.Type, "#viewRecord") &&
		strings.Contains(post.Post.Embed.Record.Value.Type, "post")
}

func getQuotePostAuthor(post bSkyPost) string {
	if hasEmbedPost(post) {
		handle := post.Post.Embed.Record.Author.Handle
		name := post.Post.Embed.Record.Author.DisplayName
		if name != "" {
			return name
		}
		return handle
	}
	return ""
}

func getQuotePostCreatedAt(post bSkyPost) time.Time {
	if hasEmbedPost(post) {
		t, err := time.Parse(time.RFC3339, post.Post.Embed.Record.Value.CreatedAt)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func getQuotePostText(post bSkyPost) string {
	if hasEmbedPost(post) {
		return post.Post.Embed.Record.Value.Text
	}
	return ""
}

func getQuotePostLink(post bSkyPost) (string, string, int, int) {
	embed := post.Post.Embed.Record.Value.Embed
	facets := post.Post.Embed.Record.Value.Facets
	if hasEmbedPost(post) && strings.Contains(embed.Type, "external") {
		title := embed.External.Title
		if title == "" {
			title = embed.External.URI
		}
		return embed.External.URI, title, 0, 0
	}
	if len(facets) > 0 {
		for _, facet := range facets {
			for _, feature := range facet.Features {
				if feature.URI != "" {
					return feature.URI, feature.URI, facet.Index.ByteStart, facet.Index.ByteEnd
				}
			}
		}
	}
	return "", "", 0, 0
}

func isSafeIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		isValid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':'
		if !isValid {
			return false
		}
	}
	return true
}

func imageTmpFilePath(cid string) string {
	if !isSafeIdentifier(cid) {
		return ""
	}
	return filepath.Join(imageTmpDir, cid+".jpg")
}

func inlineLink(post bSkyPost) (string, int, int) {
	for _, facet := range post.Post.Record.Facets {
		for _, feature := range facet.Features {
			if strings.Contains(feature.Type, "#link") {
				return feature.URI, facet.Index.ByteStart, facet.Index.ByteEnd
			}
		}
	}
	return "", 0, 0
}

func pruneDiskImageCache() {
	entries, err := os.ReadDir(imageTmpDir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(imageTmpDir, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		if now.Sub(info.ModTime()) > time.Hour {
			if err := os.Remove(path); err != nil {
				logError("failed to remove " + path + ": " + err.Error())
			}
		}
	}
}
