.PHONY: build install setup clean test

BINARY_NAME=ctree
TOGGLE_SCRIPT=ctree-toggle
AUTO_OPEN_SCRIPT=ctree-auto-open
INSTALL_DIR=$(HOME)/.local/bin

build:
	go build -o bin/$(BINARY_NAME) ./cmd/ctree

install: build
	mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	codesign -s - $(INSTALL_DIR)/$(BINARY_NAME) 2>/dev/null || true
	cp scripts/ctree-toggle.sh $(INSTALL_DIR)/$(TOGGLE_SCRIPT)
	cp scripts/ctree-auto-open.sh $(INSTALL_DIR)/$(AUTO_OPEN_SCRIPT)
	chmod +x $(INSTALL_DIR)/$(TOGGLE_SCRIPT) $(INSTALL_DIR)/$(AUTO_OPEN_SCRIPT)
	@echo ""
	@echo "Installed to $(INSTALL_DIR)"
	@echo "Add to ~/.tmux.conf:"
	@echo "  bind-key p run-shell \"$(INSTALL_DIR)/$(TOGGLE_SCRIPT)\""

setup: install
	$(INSTALL_DIR)/$(BINARY_NAME) setup

clean:
	rm -rf bin/ $(BINARY_NAME)

test:
	go test ./...
