package main

import (
	"bytes"
	"image"
	_ "image/png"
	"testing"
)

// maxAppIconDim mirrors go-gui's maxWindowIconDim (gui/backend/gl/icon_x11.go
// and platform_win32.go). An icon above it is rejected by the backend rather
// than scaled, so the window silently keeps the WM's default.
const maxAppIconDim = 1024

// maxAppIconBytes bounds what the embed costs every binary on every platform.
// The current kite-dock.png is ~77KB, so this leaves plenty of room for an
// artwork change while still failing loudly on an accidental 4K master.
const maxAppIconBytes = 512 * 1024

func TestAppIconDecodes(t *testing.T) {
	if len(appIconPNG) == 0 {
		t.Fatal("appIconPNG is empty: the //go:embed in icon.go did not resolve")
	}
	if len(appIconPNG) > maxAppIconBytes {
		t.Errorf("appIconPNG is %d bytes, want <= %d", len(appIconPNG), maxAppIconBytes)
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(appIconPNG))
	if err != nil {
		t.Fatalf("decode appIconPNG: %v", err)
	}
	if format != "png" {
		t.Errorf("format: got %q, want png — IconPNG must be PNG-encoded", format)
	}
	if cfg.Width != cfg.Height {
		t.Errorf("not square: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Width > maxAppIconDim || cfg.Height > maxAppIconDim {
		t.Errorf("too large: %dx%d, want <= %d per side", cfg.Width, cfg.Height, maxAppIconDim)
	}
}

// TestWindowCfgWiring guards the regression described in icon.go: an empty
// IconPNG shows go-gui's default icon in the Dock and Cmd-Tab even from a
// correctly bundled build, and no error is raised anywhere.
func TestWindowCfgWiring(t *testing.T) {
	cfg := kiteWindowCfg(&App{ShowImages: true})

	if len(cfg.IconPNG) == 0 {
		t.Error("IconPNG is empty: the app will show go-gui's default icon")
	}
	if !bytes.Equal(cfg.IconPNG, appIconPNG) {
		t.Error("IconPNG is not appIconPNG")
	}
	// Must match StartupWMClass in assets/go-kite.desktop, or Linux taskbars
	// cannot associate the window with the installed hicolor icon.
	if cfg.WMClass != "go-kite" {
		t.Errorf("WMClass: got %q, want %q", cfg.WMClass, "go-kite")
	}
	if cfg.OnInit == nil {
		t.Error("OnInit is nil: the app would never load a session")
	}
}
