.PHONY: all clean icons dist dist-macos dist-linux dist-windows \
	test test-race vet lint build prepush

KITE_BIN     := go-kite
APP_NAME     := Kite
# Pre-built .icns (see assets/icon/README.md); buildapp copies it into the
# bundle and points CFBundleIconFile at it.
APP_ICON     := assets/icon/kite.icns
# buildapp comes from the go-gui module graph — go.mod's pin — not from a
# sibling working copy. A sibling path makes `make all` fail outright on a
# fresh clone, and silently bundle with a different tool than the pinned
# version when the checkout is stale.
BUILDAPP_PKG := github.com/go-gui-org/go-gui/cmd/buildapp
BUILDAPP_BIN := build/buildapp
# Code-signing identity for the bundle. Empty (the default) means buildapp
# signs ad-hoc, and an ad-hoc signature has no certificate for TCC to key a
# permission grant against — TCC falls back to the cdhash, which changes on
# every build, so each rebuild silently revokes any granted permission while
# System Settings keeps showing it as granted. Set this to a self-signed
# code-signing certificate to keep grants across rebuilds:
#   make SIGN_IDENTITY="My Dev Cert"
# BUILDAPP_SIGN_IDENTITY in the environment does the same without the flag.
SIGN_IDENTITY ?=
SIGN_FLAG    := $(if $(SIGN_IDENTITY),-sign "$(SIGN_IDENTITY)",)

# Gate recipes resolve modules from go.mod, not from a go.work workspace.
# CI never sees a workspace file, so a gate that used one would answer a
# different question than "will CI go green". The app build targets below
# deliberately keep a bare `go` so local development against a sibling
# go-gui checkout still works.
GO := GOWORK=off go

# The toolchain the *app* build targets use, as opposed to the gate targets
# above. Bare `go` by default so local development against sibling go-gui and
# go-glyph checkouts still works (see the comment above). The dist targets
# override it, because a shipped artifact must be built from the versions
# go.sum records rather than from whatever is checked out next door.
GO_APP := go

# golangci-lint is its own binary, so $(GO) does not cover it — but it
# honours go.work the same way the toolchain does. Without GOWORK=off it
# would type-check against sibling working copies and report breakage that
# CI, which builds the pinned versions, will never see.
LINT := GOWORK=off golangci-lint

all: $(APP_NAME).app

# Depends on the embedded dock PNG (icon.go's //go:embed) as well as the
# sources: `make icons` regenerates it, and without this dependency the
# binary would keep the old artwork while the re-bundled .app carried the
# new .icns — Finder and the Dock would disagree about what Kite looks like.
$(KITE_BIN): *.go go.mod go.sum assets/icon/kite-dock.png
	# -tags=prod disables go-gui's F12 dev inspector in the shipped app;
	# -trimpath keeps the binary reproducible; -ldflags "-s -w" strips
	# the symbol table and DWARF, shrinking the binary ~30% (crash stacks
	# lose function names as a tradeoff).
	$(GO_APP) build -tags=prod -trimpath -ldflags="-s -w" -o $@ .

$(BUILDAPP_BIN):
	mkdir -p build
	$(GO_APP) build -o $@ $(BUILDAPP_PKG)

# Depends on the icon so swapping artwork forces a re-bundle; Go source
# changes are caught by go build itself, not by make's timestamp check.
$(APP_NAME).app: $(KITE_BIN) $(BUILDAPP_BIN) $(APP_ICON)
	$(BUILDAPP_BIN) -bundle-deps -o . -name $(APP_NAME) \
		-id github.com.go-gui-org.go-kite -icon $(APP_ICON) \
		$(SIGN_FLAG) $(KITE_BIN)

# Regenerate every icon artifact from assets/icon/kite.svg. Deliberately not
# part of prepush: it needs rsvg-convert, ImageMagick and iconutil, which CI
# has none of. The outputs are committed precisely so CI never runs this.
icons:
	./assets/icon/generate.sh

# Distributable packages, written to dist/ with a SHA256SUMS covering them.
#
#   make dist                     everything this host can build
#   make dist VERSION=1.2.0       stamp a real version instead of git describe
#   make dist-linux ARCHES=amd64  one architecture instead of amd64 + arm64
#
# Linux and Windows cross-build from any host: go-gui's gl backend is CGo-free
# on both, so no cross toolchain is involved. macOS is the exception -- its
# backend is ObjC behind cgo, so dist-macos only runs on macOS, and only for
# the host architecture.
#
# All three go through packaging/package.sh; the logic is shell, not make,
# because it is a sequence of steps rather than a dependency graph.
dist-macos dist-linux dist-windows: GO_APP := GOWORK=off go

# Depends on the bundle, so the .dmg can never ship a stale Kite.app.
dist-macos: $(APP_NAME).app
	./packaging/package.sh macos

dist-linux:
	./packaging/package.sh linux

dist-windows:
	./packaging/package.sh windows

# dist-macos is left out on non-Darwin hosts rather than failing the build.
dist: dist-linux dist-windows $(if $(filter Darwin,$(shell uname -s)),dist-macos)

# Run the test suite. Mirrors the CI test job's non-race half (macOS runner).
test:
	$(GO) test ./...

# Race-enabled tests. CI runs -race on its Linux runner only; running it
# here covers that leg from any host.
test-race:
	$(GO) test -race -count=1 ./...

# Static analysis. Mirrors the CI vet job.
vet:
	$(GO) vet ./...

# Lint. CI uses golangci-lint-action without a pinned version and this repo
# carries no .golangci.yml, so both CI and this target run the golangci-lint
# defaults. Keep it unpinned so the two stay in agreement.
lint:
	$(LINT) run ./...

build:
	$(GO) build ./...

# Recommended full local validation before pushing (issue go-gui#314).
# Approximates the CI matrix from one host: race tests, vet, lint, build.
# Aborts on the first failing target.
#
# Omissions vs CI, by design: the OS matrix itself — CI runs the suite on
# both ubuntu-latest and macos-latest, and only the host's own platform is
# exercised here.
prepush: test-race vet lint build

clean:
	rm -f $(KITE_BIN)
	rm -rf $(APP_NAME).app
	rm -rf build
	rm -rf dist
