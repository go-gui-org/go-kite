# go_kite

A desktop Bluesky client written in Go using [`go-gui`](https://github.com/go-gui-org/go-gui).

This repo ports the original V-based Kite app to the new Go GUI framework with feature parity: login, session reuse, timeline polling, links, quoted posts, images, and keyboard font-size shortcuts.

## Features

- Login with Bluesky credentials
- Session persistence in `~/.kite.toml`
- Automatic timeline refresh loop
- Retry + session refresh fallback on API failures
- Quoted post rendering
- Clickable links (opened via native platform handlers)
- Optional image loading with local cache
- Font size controls: `Alt+Up` and `Alt+Down`
- Right-click timeline to jump to top

## Run

```bash
go mod tidy
go run .
```

Disable image downloads/rendering:

```bash
go run . -no-images
```

## Test

```bash
go test ./...
```

## Project Layout

- `main.go` - app entry point and window lifecycle
- `views.go` - login and timeline UI views
- `timeline.go` - timeline loop, post conversion, image cache/download logic
- `api.go` - Bluesky API client
- `session.go` - session load/save/refresh
- `textutil.go` - formatting, sanitization, and shared helpers
- `models.go` - app, timeline, and API data models

## Notes

- Session file path is intentionally compatible with the original app: `~/.kite.toml`.
- Cached/resized timeline images are stored under your temp directory at `.../kite`.
