package main

import (
	"math"
	"testing"

	"github.com/go-gui-org/go-gui/gui"
)

func TestScaledImageDimsDefaultsDegenerateInputs(t *testing.T) {
	w, h := scaledImageDims(0, 0)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("zero dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(-10, -5)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("negative dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(500, 0)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("zero height: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(0, 500)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("zero width: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
}

func TestScaledImageDimsScalesWideImages(t *testing.T) {
	// 640x480 → 270 wide, height kept in proportion (480*270/640).
	w, h := scaledImageDims(640, 480)
	if w != imageWidth {
		t.Fatalf("width: got %v, want %v", w, imageWidth)
	}
	if want := float32(480) * (float32(imageWidth) / 640); h != want {
		t.Fatalf("height: got %v, want %v", h, want)
	}
}

func TestScaledImageDimsCapsHeight(t *testing.T) {
	// Portrait embed taller than the cap, both before and after scaling.
	if _, h := scaledImageDims(imageWidth, 800); h != maxImageHeight {
		t.Fatalf("height: got %v, want %v", h, maxImageHeight)
	}
	if _, h := scaledImageDims(640, 3000); h != maxImageHeight {
		t.Fatalf("scaled height: got %v, want %v", h, maxImageHeight)
	}
}

func TestScaledImageDimsFloorsSubPixelHeight(t *testing.T) {
	// Absurdly wide embed scales height below 1px; floor keeps it visible.
	w, h := scaledImageDims(1e9, 10)
	if w != imageWidth {
		t.Fatalf("width: got %v, want %v", w, imageWidth)
	}
	if h != 1 {
		t.Fatalf("height: got %v, want 1", h)
	}
}

func TestScaledImageDimsRejectsNaNAndInf(t *testing.T) {
	// Every comparison against NaN is false, so without the up-front
	// guard a NaN dim sails through all four branches unchanged and
	// poisons the layout with NaN extents.
	w, h := scaledImageDims(float32(math.NaN()), float32(math.NaN()))
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("NaN dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	w, h = scaledImageDims(float32(math.NaN()), 100)
	if w != imageWidth || h != 100 {
		t.Fatalf("NaN width: got %vx%v, want %vx%v", w, h, imageWidth, 100)
	}
	// +Inf width scales the height to zero, which then falls into the
	// display default rather than returning a vanishing embed.
	w, h = scaledImageDims(float32(math.Inf(1)), 480)
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("+Inf width: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
	_, h = scaledImageDims(640, float32(math.Inf(1)))
	if h != maxImageHeight {
		t.Fatalf("+Inf height: got %v, want %v", h, maxImageHeight)
	}
	// -Inf is caught by the plain <= 0 branches.
	w, h = scaledImageDims(float32(math.Inf(-1)), float32(math.Inf(-1)))
	if w != imageWidth || h != maxImageHeight {
		t.Fatalf("-Inf dims: got %vx%v, want %vx%v", w, h, imageWidth, maxImageHeight)
	}
}

// TestLoginReady pins the one rule the Submit button and Enter in
// either login field share. Enter fires OnEnter even while the button
// is disabled, so submitLogin must re-check it.
func TestLoginReady(t *testing.T) {
	cases := []struct {
		name string
		app  App
		want bool
	}{
		{"both filled", App{UserName: "a", Password: "b"}, true},
		{"empty user", App{Password: "b"}, false},
		{"empty password", App{UserName: "a"}, false},
		{"whitespace only", App{UserName: " \t", Password: "b"}, false},
		{"login in flight", App{UserName: "a", Password: "b", LoginPending: true}, false},
	}
	for _, tc := range cases {
		if got := loginReady(&tc.app); got != tc.want {
			t.Errorf("%s: loginReady = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestSafeLinkRun pins the guard textLink applies before it hands a
// run to RTF. Post URIs come from the network, and RTF would also act
// on mailto and in-page anchors, so only http and https keep a Link.
func TestSafeLinkRun(t *testing.T) {
	cases := []struct {
		link string
		want string
	}{
		{"https://bsky.app/profile/a", "https://bsky.app/profile/a"},
		{"HTTP://example.com", "HTTP://example.com"},
		{"javascript:alert(1)", ""},
		{"mailto:a@example.com", ""},
		{"#top", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := safeLinkRun(gui.RichTextRun{Text: "t", Link: c.link})
		if got.Link != c.want {
			t.Errorf("safeLinkRun(%q).Link = %q, want %q", c.link, got.Link, c.want)
		}
		if got.Text != "t" {
			t.Errorf("safeLinkRun(%q) changed Text to %q", c.link, got.Text)
		}
	}
}
