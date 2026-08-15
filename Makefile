.PHONY: all clean test test-race vet lint build prepush

KITE_BIN     := go-kite
APP_NAME     := Kite
BUILDAPP_DIR := ../go-gui/cmd/buildapp
BUILDAPP_BIN := $(BUILDAPP_DIR)/buildapp

# Gate recipes resolve modules from go.mod, not from a go.work workspace.
# CI never sees a workspace file, so a gate that used one would answer a
# different question than "will CI go green". The app build targets below
# deliberately keep a bare `go` so local development against a sibling
# go-gui checkout still works.
GO := GOWORK=off go

# golangci-lint is its own binary, so $(GO) does not cover it — but it
# honours go.work the same way the toolchain does. Without GOWORK=off it
# would type-check against sibling working copies and report breakage that
# CI, which builds the pinned versions, will never see.
LINT := GOWORK=off golangci-lint

all: $(APP_NAME).app

$(KITE_BIN): *.go go.mod go.sum
	# -tags=prod disables go-gui's F12 dev inspector in the shipped app;
	# -trimpath keeps the binary reproducible; -ldflags "-s -w" strips
	# the symbol table and DWARF, shrinking the binary ~30% (crash stacks
	# lose function names as a tradeoff).
	go build -tags=prod -trimpath -ldflags="-s -w" -o $@ .

$(BUILDAPP_BIN):
	cd $(BUILDAPP_DIR) && go build -o buildapp .

$(APP_NAME).app: $(KITE_BIN) $(BUILDAPP_BIN)
	$(BUILDAPP_BIN) -bundle-deps -o . -name $(APP_NAME) \
		-id github.com.go-gui-org.go-kite $(KITE_BIN)

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
	cd $(BUILDAPP_DIR) && rm -f buildapp
