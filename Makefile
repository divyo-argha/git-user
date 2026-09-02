BINARY   := git-user
BUILD_DIR := dist
VERSION  := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
DATE     := $(shell date -u +'%Y-%m-%d')

.PHONY: build install install-local uninstall clean test release-test release-test-check

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w -X main.buildVersion=$(VERSION) -X main.date=$(DATE)" -o $(BUILD_DIR)/$(BINARY) ./cmd/git-user
	@echo "Binary built: $(BUILD_DIR)/$(BINARY)"

# Install to /usr/local/bin so 'git user' works as a subcommand
install: build
	@install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@ln -sf /usr/local/bin/$(BINARY) /usr/local/bin/gu
	@echo "Installed to /usr/local/bin/$(BINARY) (and alias /usr/local/bin/gu)"
	@echo "You can now run: git user <command>, git-user <command>, or gu <command>"

# Install to ~/bin (no sudo required)
install-local: build
	@mkdir -p $$HOME/bin
	@install -m 755 $(BUILD_DIR)/$(BINARY) $$HOME/bin/$(BINARY)
	@ln -sf $$HOME/bin/$(BINARY) $$HOME/bin/gu
	@echo "Installed to $$HOME/bin/$(BINARY) (and alias $$HOME/bin/gu)"
	@echo "Make sure $$HOME/bin is on your PATH."

uninstall:
	@rm -f /usr/local/bin/$(BINARY) /usr/local/bin/gu
	@echo "Uninstalled."

clean:
	@rm -rf $(BUILD_DIR)

test:
	@go run test/runner/main.go

release-test:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser is not installed."; \
		echo "Install the prebuilt binary from https://github.com/goreleaser/goreleaser/releases"; \
		echo "(or: brew install goreleaser / go install github.com/goreleaser/goreleaser/v2@latest)."; \
		exit 1; \
	fi

release-test-check:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
	else \
		echo "goreleaser is not installed — skipping config check."; \
	fi

