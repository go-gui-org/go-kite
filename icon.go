package main

import _ "embed"

// appIconPNG is the icon handed to gui.WindowCfg.IconPNG.
//
// This is not redundant with Kite.app's kite.icns. The macOS backend calls
// -[NSApplication setApplicationIconImage:] during window creation, and falls
// back to go-gui's own default icon when IconPNG is empty. That runtime call
// wins over CFBundleIconFile for the running process, so leaving this unset
// installs the go-gui icon over Kite's on every launch, even from a correctly
// bundled build. The symptom is the go-gui icon in the Dock and Cmd-Tab while
// Finder still shows Kite — which reads exactly like a stale LaunchServices
// icon cache, and flushing the cache will not fix it. TestWindowCfgWiring
// fails if the field is dropped.
//
// The two cover cases the other cannot:
//
//   - kite.icns / CFBundleIconFile — Finder, the Dock's persistent tile, Get
//     Info. Applies only to a bundled build.
//   - IconPNG — the running process's application icon on macOS, the
//     taskbar/alt-tab window icon on Windows and X11, and the only icon that
//     exists at all under `go run .`, where there is no bundle.
//
// 512x512 with the artwork inset to 824/1024 of the canvas, matching the
// padding of the .icns slots so the Dock tile lines up with its neighbours.
//
//go:embed assets/icon/kite-dock.png
var appIconPNG []byte
