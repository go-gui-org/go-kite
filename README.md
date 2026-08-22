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

## Icons

`assets/logo.svg` is the artwork. Everything shipped is rendered from the two
derived masters beside it:

```bash
make icons
```

That regenerates `assets/icon/kite.icns` (macOS bundle), `assets/icon/kite.ico`
plus `rsrc_windows_*.syso` (Windows executable resource), and
`assets/icons/hicolor/**` (Linux). The outputs are committed, so a build never
needs the rendering tools; `make icons` needs `rsvg-convert`, ImageMagick and
`iconutil`.

The icon is wired in two independent places and both are required — the `.icns`
covers Finder and the Dock tile of a bundled build, while
`gui.WindowCfg.IconPNG` covers the running process and is the only icon that
exists under `go run .`. See `assets/icon/README.md` for why, and for the Linux
install paths.

## Packaging

```bash
make dist                     # every package this host can build, into dist/
make dist VERSION=1.2.0       # stamp a real version instead of `git describe`
make dist-linux ARCHES=amd64  # one architecture instead of amd64 + arm64
make dist-macos               # .dmg only (macOS host)
```

| Target         | Produces                                                        |
| -------------- | --------------------------------------------------------------- |
| `dist-macos`   | `Kite-<ver>-macos-<arch>.dmg`                                   |
| `dist-linux`   | `go-kite-<ver>-linux-<arch>.tar.gz`, `go-kite_<ver>_<arch>.deb` |
| `dist-windows` | `go-kite-<ver>-windows-<arch>.zip`                              |

Linux and Windows cross-build from any host with no cross toolchain, because
go-gui's `gl` backend is CGo-free on both — it dlopens `libEGL`/`opengl32`
through purego, and go-glyph supplies a pure-Go text pipeline when cgo is off.
`CGO_ENABLED=0` is forced even for a native Linux build, so the binary links no
glibc and runs on any distro; the only runtime dependency is `libEGL.so.1`
(`libegl1`), which is what the `.deb` declares.

macOS is the exception. Its backend is Objective-C behind cgo, so `dist-macos`
runs only on macOS and only for the host architecture. The `.dmg` is built from
`Kite.app`, which `make all` produces, and carries whatever signature
`SIGN_IDENTITY` gave it — see the Makefile. Nothing here notarizes, so a
downloaded `.dmg` still hits Gatekeeper on another Mac.

The `.deb` is assembled directly from `ar` plus two tarballs rather than by
`dpkg-deb`, which is what lets a macOS host build Debian packages at all. The
tarball is the fallback for everything else: unpack it and run its `install.sh`
(`PREFIX` defaults to `~/.local`).

Release builds always pin to `go.mod`, never to a `go.work` pointing at sibling
checkouts. `dist/` is gitignored; `dist/SHA256SUMS` covers every artifact and is
rewritten on each run.

## Project Layout

- `main.go` - app entry point and window lifecycle
- `views.go` - login and timeline UI views
- `timeline.go` - timeline loop, post conversion, image cache/download logic
- `api.go` - Bluesky API client
- `session.go` - session load/save/refresh
- `textutil.go` - formatting, sanitization, and shared helpers
- `models.go` - app, timeline, and API data models
- `icon.go` - the embedded window/Dock icon, and why it is not the `.icns`
- `packaging/package.sh` - builds the macOS, Linux and Windows packages

## Notes

- Session file path is intentionally compatible with the original app:
  `~/.kite.toml`.
- Cached/resized timeline images are stored under your temp directory at
  `.../kite`.

## License

MIT — see [LICENSE](LICENSE).
