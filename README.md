# go_kite

A desktop Bluesky client written in Go using
[`go-gui`](https://github.com/go-gui-org/go-gui).

This repo ports the original V-based Kite app to the new Go GUI framework with
feature parity: login, session reuse, timeline polling, links, quoted posts,
images, and keyboard font-size shortcuts.

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

Run the full local validation gate before pushing a branch:

```bash
make prepush
```

`make prepush` approximates this repo's CI from one host: race-enabled tests,
`go vet`, lint, and a build. It aborts on the first failing target. Individual
targets (`make test`, `test-race`, `vet`, `lint`, `build`) are available for a
tighter loop while iterating.

Gate targets run with `GOWORK=off` so they resolve the versions in `go.mod`,
which is what CI does — a local `go.work` pointing at a sibling checkout would
otherwise validate something CI never sees. The app build targets (`make all`)
keep using the workspace.

### CI-only validation

`make prepush` covers one host. CI additionally runs the whole suite on both
`ubuntu-latest` and `macos-latest`, so a platform-specific failure on the OS you
are not using can only be caught there.

## Project Layout

- `main.go` - app entry point and window lifecycle
- `views.go` - login and timeline UI views
- `timeline.go` - timeline loop, post conversion, image cache/download logic
- `api.go` - Bluesky API client
- `session.go` - session load/save/refresh
- `textutil.go` - formatting, sanitization, and shared helpers
- `models.go` - app, timeline, and API data models

## Notes

- Session file path is intentionally compatible with the original app:
  `~/.kite.toml`.
- Cached/resized timeline images are stored under your temp directory at
  `.../kite`.
