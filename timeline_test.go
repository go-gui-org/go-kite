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
