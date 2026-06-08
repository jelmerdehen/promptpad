# promptpad — build + install.
# Run `make install` once; from then on i3 just calls $(PREFIX)/bin/promptpad.

PREFIX     ?= $(HOME)/.local
BIN_DIR    ?= $(PREFIX)/bin
INSTALL    ?= install
GO         ?= go
BINARY     := bin/promptpad
PKG        := ./cmd/promptpad
LDFLAGS    := -s -w
BUILDFLAGS := -trimpath -ldflags='$(LDFLAGS)'

.PHONY: all build install uninstall reinstall clean doctor run test fmt vet

all: build

build: $(BINARY)

$(BINARY): $(shell find cmd internal -name '*.go') go.mod go.sum
	$(GO) build $(BUILDFLAGS) -o $(BINARY) $(PKG)

install: build
	$(INSTALL) -d $(BIN_DIR)
	# Symlink (not copy) so the binary's EvalSymlinks finds the repo
	# and resolves snippets/ relative to its real location.
	ln -sfn $(CURDIR)/$(BINARY) $(BIN_DIR)/promptpad
	@echo "Installed symlink: $(BIN_DIR)/promptpad -> $(CURDIR)/$(BINARY)"
	@command -v promptpad >/dev/null 2>&1 || echo "Note: $(BIN_DIR) not in PATH"

uninstall:
	rm -f $(BIN_DIR)/promptpad
	@echo "Removed: $(BIN_DIR)/promptpad"

reinstall: uninstall install

clean:
	rm -f $(BINARY)

doctor: build
	./$(BINARY) doctor

run: build
	./$(BINARY) $(ARGS)

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...
