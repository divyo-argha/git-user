BINARY   := git-user
BUILD_DIR := dist

.PHONY: build install clean test

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) .
	@echo "Binary built: $(BUILD_DIR)/$(BINARY)"

# Install to /usr/local/bin so 'git user' works as a subcommand
install: build
	@install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"
	@echo "You can now run: git user <command>"

# Install to ~/bin (no sudo required)
install-local: build
	@mkdir -p $$HOME/bin
	@install -m 755 $(BUILD_DIR)/$(BINARY) $$HOME/bin/$(BINARY)
	@echo "Installed to $$HOME/bin/$(BINARY)"
	@echo "Make sure $$HOME/bin is on your PATH."

uninstall:
	@rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstalled."

clean:
	@rm -rf $(BUILD_DIR)

test:
	go test ./...
