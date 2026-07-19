package main

import "testing"

func TestFromBlueskyTimelineFiltersRepliesAndCaps(t *testing.T) {
	timeline := bSkyTimeline{Posts: make([]bSkyPost, 0, 5)}
	for i := 0; i < 5; i++ {
		p := minimalPost("text")
		if i == 1 {
			p.Post.Record.Reply.Parent.CID = "parent"
		}
		timeline.Posts = append(timeline.Posts, p)
	}

	got := fromBlueskyTimeline(timeline, 3)
	if len(got.Posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(got.Posts))
	}
}

func TestFromBlueskyPostRemovesInlineLink(t *testing.T) {
	p := minimalPost("hello https://example.com world")
	p.Post.Record.Facets = []bSkyFacet{{}}
	p.Post.Record.Facets[0].Features = []struct {
		Type string `json:"$type"`
		URI  string `json:"uri"`
	}{{Type: "app.bsky.richtext.facet#link", URI: "https://example.com"}}
	p.Post.Record.Facets[0].Index.ByteStart = 6
	p.Post.Record.Facets[0].Index.ByteEnd = 25

	got := fromBlueskyPost(p)
	if got.LinkURI != "https://example.com" {
		t.Fatalf("expected extracted URI, got %q", got.LinkURI)
	}
	if got.FormattedText == "hello https://example.com world" {
		t.Fatalf("expected inline URI removal")
	}
}

func TestGetQuotePostLinkUsesMatchingFacetIndexes(t *testing.T) {
	p := minimalPost("text")
	p.Post.Embed.Record.Value.Facets = []bSkyFacet{
		{
			Features: []struct {
				Type string `json:"$type"`
				URI  string `json:"uri"`
			}{
				{Type: "app.bsky.richtext.facet#link", URI: ""},
			},
			Index: struct {
				ByteStart int `json:"byteStart"`
				ByteEnd   int `json:"byteEnd"`
			}{ByteStart: 1, ByteEnd: 2},
		},
		{
			Features: []struct {
				Type string `json:"$type"`
				URI  string `json:"uri"`
			}{
				{Type: "app.bsky.richtext.facet#link", URI: "https://example.com/quote"},
			},
			Index: struct {
				ByteStart int `json:"byteStart"`
				ByteEnd   int `json:"byteEnd"`
			}{ByteStart: 6, ByteEnd: 11},
		},
	}

	uri, _, start, end := getQuotePostLink(p)
	if uri != "https://example.com/quote" {
		t.Fatalf("unexpected uri: %q", uri)
	}
	if start != 6 || end != 11 {
		t.Fatalf("unexpected indexes: start=%d end=%d", start, end)
	}
}

func minimalPost(text string) bSkyPost {
	var p bSkyPost
	p.Post.URI = "at://did:plc:abc/app.bsky.feed.post/123"
	p.Post.Author.Handle = "alice.bsky.social"
	p.Post.Author.DisplayName = "Alice"
	p.Post.Author.DID = "did:plc:abc"
	p.Post.Record.Text = text
	p.Post.Record.CreatedAt = "2025-01-01T00:00:00Z"
	return p
}

func revealPost(id, repostBy string) Post {
	return Post{ID: id, RepostBy: repostBy, FormattedText: "text"}
}

func TestSetRevealAnchorOnPrepend(t *testing.T) {
	app := &App{Timeline: Timeline{Posts: []Post{revealPost("a", "")}}}
	incoming := Timeline{Posts: []Post{revealPost("b", ""), revealPost("a", "")}}

	setRevealAnchor(app, incoming)
	if app.RevealAnchorID != postViewID(revealPost("a", "")) {
		t.Fatalf("unexpected anchor: %q", app.RevealAnchorID)
	}
}

func TestSetRevealAnchorNoChange(t *testing.T) {
	app := &App{Timeline: Timeline{Posts: []Post{revealPost("a", "")}}}
	incoming := Timeline{Posts: []Post{revealPost("a", "")}}

	setRevealAnchor(app, incoming)
	if app.RevealAnchorID != "" {
		t.Fatalf("anchor set on unchanged timeline: %q", app.RevealAnchorID)
	}
}

func TestSetRevealAnchorInitialLoad(t *testing.T) {
	app := &App{}
	incoming := Timeline{Posts: []Post{revealPost("a", "")}}

	setRevealAnchor(app, incoming)
	if app.RevealAnchorID != "" {
		t.Fatalf("anchor set on initial load: %q", app.RevealAnchorID)
	}
}

func TestFirstRenderedPostIDSkipsEmptyPosts(t *testing.T) {
	tl := Timeline{Posts: []Post{
		{ID: "empty", FormattedText: "  "},
		revealPost("a", ""),
	}}
	if got := firstRenderedPostID(tl); got != postViewID(revealPost("a", "")) {
		t.Fatalf("unexpected first rendered id: %q", got)
	}
	if got := firstRenderedPostID(Timeline{}); got != "" {
		t.Fatalf("expected empty id for empty timeline, got %q", got)
	}
}

func TestPostViewIDDisambiguatesReposts(t *testing.T) {
	if postViewID(revealPost("a", "")) == postViewID(revealPost("a", "bob")) {
		t.Fatal("post and its repost must have distinct view IDs")
	}
}

func TestPostIsRendered(t *testing.T) {
	if postIsRendered(Post{}) {
		t.Fatal("empty post should not be rendered")
	}
	if postIsRendered(Post{FormattedText: "  "}) {
		t.Fatal("post with whitespace-only FormattedText should not be rendered")
	}
	if !postIsRendered(Post{FormattedText: "hello"}) {
		t.Fatal("post with FormattedText should be rendered")
	}
	if !postIsRendered(Post{FormattedQuoteText: "quote"}) {
		t.Fatal("post with only FormattedQuoteText should be rendered")
	}
}

func TestSetRevealAnchorPostsReplaced(t *testing.T) {
	app := &App{Timeline: Timeline{Posts: []Post{revealPost("old", "")}}}
	incoming := Timeline{Posts: []Post{revealPost("new", "")}}

	setRevealAnchor(app, incoming)
	if app.RevealAnchorID != postViewID(revealPost("old", "")) {
		t.Fatalf("anchor should be set when old post is replaced: %q", app.RevealAnchorID)
	}
}
