# The version string comes from the git tag, never from a constant in source.
# `git describe` gives v0.1.0 at the tag and v0.1.0-3-gabc1234 three commits
# later. `git tag vX.Y.Z` is the single source of truth.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

# go.mod pins a toolchain and the go command re-execs into it. A GOROOT
# exported from a different SDK contradicts that: the re-exec'd compiler and
# GOROOT then disagree, and `go test -race` rebuilds the standard library
# straight into the mismatch. `unexport` does not strip an inherited
# environment variable, so every toolchain invocation goes through this
# prefix and the go.mod directive stays the only source of truth for which
# toolchain runs.
GO := env -u GOROOT go
GOVULNCHECK := env -u GOROOT govulncheck

.PHONY: check fmt fmt-check vet lint test laws replay self decisions hooks replay-gate links vuln tidy-check build install git-hooks version

## check: what CI runs and what the pre-commit hook runs. Green or nothing ships.
check: fmt-check vet lint test replay self hooks links vuln tidy-check

## self: germline checks its own four artifacts with its own checker, then the
## sample system's. The first system germline governs is germline; if the
## checker cannot pass its own repository it has no business in anyone else's.
##
## The brownfield case is checked where it lives: the wovim pilot moved into
## the fork's own repository, so `make check` there is what says germline can
## pass a 363 KLOC C system. This repository is the method, not the evidence.
self: build
	bin/germline check
	bin/germline check -C examples/querystring

## links: every relative .md path a tracked markdown file mentions must exist
links:
	scripts/check-links.sh

## fmt: rewrite files in place with the same formatters CI checks
fmt:
	golangci-lint fmt ./...

## fmt-check: fail if any file would change under gofumpt/goimports
fmt-check:
	golangci-lint fmt --diff ./...

vet:
	$(GO) vet ./...

## lint: golangci-lint with .golangci.yml; staticcheck runs inside it with ST on
lint:
	golangci-lint run ./...

## test: race detector on, coverage summary printed
test:
	$(GO) test -race -cover ./...

## laws: the property-based laws only
laws:
	$(GO) test -race ./laws/...

## replay: the laws, then whatever of the loop runs without two versions to
## compare. A real replay needs both, so it is a command and not a target:
##   bin/germline replay -was '0.1.0=<cmd>' -now '0.2.0=<cmd>'
replay: laws
	@if [ -z "$$(ls -A corpus/inputs 2>/dev/null | grep -v '^\.gitkeep$$')" ]; then \
		echo "replay: corpus/inputs is empty; nothing recorded yet, laws only"; \
	else \
		echo "replay: $$(ls corpus/inputs | grep -vc '^\.gitkeep$$') inputs recorded;"; \
		echo "        name two versions to compare them:"; \
		echo "        bin/germline replay -was '<v>=<cmd>' -now '<v>=<cmd>'"; \
	fi

## decisions: print the external decision store's current decisions, each
## with what it superseded. Maintainer-only: needs winze-query and
## `git config winze.store`.
STORE := $(shell git config --get winze.store)
decisions:
	@test -n "$(STORE)" || { echo "no winze.store configured; run: git config winze.store <dir>"; exit 1; }
	@winze-query --decisions "$(STORE)"

## boundaries: project the decision record's declined promises into each
## system's BOUNDARY.md. The record is the source; `germline check` refuses a
## file that has drifted from it, so this is the fix rather than the check.
##
## One record serves every system that points at it, so a system living in
## another repository regenerates from the same store with its own
## `make boundaries` there.
boundaries: build
	bin/germline boundary -write
	bin/germline boundary -write -C examples/querystring

## hooks: the stull machine compiles, and the Stop-hook binary is built where
## the wiring in settings.local.json points. A hook whose command does not
## exist is a session that cannot stop, so this is part of `check`.
hooks:
	$(GO) build ./hooks/...
	$(GO) build -o bin/replaygate ./hooks/replaygate/cmd/replaygate

## replay-gate: wire the Stop hook into this clone. Merges into
## .claude/settings.local.json, which is gitignored because it names an
## absolute path on this machine — so this target is the tracked form of the
## wiring, and re-running it is a no-op. Without -write the binary prints the
## merged file instead; run `bin/replaygate install` to see it first.
replay-gate: hooks
	bin/replaygate install -write

## vuln: known-vulnerability scan of the module graph
vuln:
	$(GOVULNCHECK) ./...

## tidy-check: `go mod tidy` changes nothing. It compares before against
## after, not against HEAD: diffing against HEAD reports any uncommitted
## dependency change as untidy, which is a different thing and made the gate
## red for the correct edit that added a toolchain directive.
tidy-check:
	@tmp=$$(mktemp -d) && cp go.mod go.sum "$$tmp/" && \
	 $(GO) mod tidy && \
	 status=0 && \
	 { diff -u "$$tmp/go.mod" go.mod || status=1; } && \
	 { diff -u "$$tmp/go.sum" go.sum || status=1; } ; \
	 cp "$$tmp/go.mod" "$$tmp/go.sum" . && rm -rf "$$tmp" && \
	 { [ $$status -eq 0 ] || { echo "run: go mod tidy"; exit 1; }; }

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/germline ./cmd/germline

install:
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/germline

## git-hooks: point this clone at the tracked pre-commit hook
git-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit

version:
	@echo $(VERSION)
