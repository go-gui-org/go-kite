# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## What this is

Desktop Bluesky client. Single `main` package at the repo root, built on
[`go-gui`](https://github.com/go-gui-org/go-gui) (immediate-mode retained-view
GUI, SDL2 backend, cgo). Port of the original V-based Kite app.

## Commands

```fish
make prepush          # full local gate: test-race, vet, lint, build. Run before pushing.
make test             # go test ./...
make test-race        # go test -race -count=1 ./...
make vet lint build   # individual gate targets
make all              # build Kite.app bundle (buildapp built from go.mod's pin)
go run .              # run the app
go run . -no-images   # run without image download/rendering
```

Single test: `GOWORK=off go test -run TestScaledImageDims ./...` — mirror the
gate's `GOWORK=off` so the test resolves `go.mod` versions, not sibling
checkouts.

macOS app bundle signing: `make SIGN_IDENTITY="My Dev Cert"`. Ad-hoc signing
makes TCC key permission grants to the cdhash, which changes every build and
silently revokes granted permissions. See the Makefile comment.

### GOWORK=off is deliberate

All gate targets (`test`, `test-race`, `vet`, `lint`, `build`) force
`GOWORK=off`. CI never sees a workspace file, so a gate that honored `go.work`
would validate code CI never builds. The app build targets (`$(KITE_BIN)`,
`make all`) intentionally keep a bare `go` so local development against sibling
`../go-glyph` and `../go-gui` checkouts still works.

`go.work` is gitignored; the checked-in `go.work` at the root is a template —
comments must use `//`, the parser rejects `#`.

`make prepush` covers only the host OS. CI runs the suite on both
`ubuntu-latest` and `macos-latest`, so platform-specific failures on the other
OS surface only in CI. No `.golangci.yml` exists — both CI and `make lint` run
golangci-lint defaults, unpinned, so they stay in agreement.

## Architecture

### Threading model

`go-gui` owns the UI thread. Everything touching window or app state from a
goroutine must go through `w.QueueCommand(func(w *gui.Window){...})`, then call
`w.UpdateWindow()` inside it. This is the single most load-bearing rule in the
codebase — `loginAsync` and `timelineLoop` are both built around it.

App state is one `*App` reached via `gui.State[App](w)`. Views are functions
`func(*gui.Window) gui.View` swapped with
`w.UpdateView(loginView | timelineView)`.

### Timeline loop (`timeline.go`)

One goroutine per session, cancelled by closing `app.LoopCancel` (a fresh
channel per `startTimelineLoop`). Each iteration:

1. `getTimeline` → on error, up to `maxRetryAttempts` (10) session refreshes
   with quadratic backoff (`n²` seconds); after exhaustion it clears state and
   drops back to `loginView`.
2. Convert + push the text-only timeline immediately (fast first paint).
3. If images are on, download/resize them in parallel, re-convert, push again —
   `fromBlueskyTimeline` is called twice on the same raw response because
   `postImage` only reports images already on disk.
4. Sleep one minute (cancellable).

### Scroll anchoring

`anchorTimelineReveal` runs _before_ `app.Timeline` is replaced — it captures
the old top post's on-screen position. If the reader is at the top, or idle for
`idleRevealAfter` (10 min, tracked by `app.LastInteraction` in `appOnEvent`),
new posts ease into view (`ScrollAnchorReveal`); otherwise the reading position
is pinned (`ScrollAnchor`). `postViewID` = post URI + NUL + reposter, because a
timeline can hold both a post and reposts of it.

`postIsRendered` in `views.go` and the anchor helpers must agree on which posts
render — anchor math operating on posts the view skips will pick a missing ID.

### Formatting is precomputed

`Post` carries both raw fields and `Formatted*` fields. Views never format;
`fromBlueskyPost` sanitizes and truncates once per refresh so layout does no
per-frame string work. `sanitizeText` = strip control chars + truncate overlong
words.

### Image cache

Resized JPEGs live in `$TMPDIR/kite/<cid>.jpg`, scaled to `imageWidth` (270),
clipped at `maxImageHeight` (250). `pruneDiskImageCache` deletes entries older
than one hour on each poll. `imageWriteLocks` (a `sync.Map` of mutexes keyed by
path) serializes concurrent writers of the same CID.

### Security-relevant guards

Two validators exist for a reason — keep them:

- `isSafeIdentifier` gates CIDs before they become filesystem paths
  (`imageTmpFilePath` returns `""` on rejection, and callers must skip).
- `isSafeURI` gates URIs before `OpenURI` — http/https only.

### Session

`~/.kite.toml` (TOML, path intentionally compatible with the original V app).
`refreshSession` rewrites the file on every successful token refresh.

## Conventions

- Comments explain _why_, at length, especially for non-obvious tradeoffs (see
  Makefile, `anchorTimelineReveal`, `scaledImageDims`). Match that density.
- API structs in `models.go` are hand-written mirrors of atproto lexicon JSON;
  add fields there rather than inline anonymous structs.
- Bluesky replies are filtered out of the timeline entirely
  (`fromBlueskyTimeline` skips anything with a reply parent/root).
