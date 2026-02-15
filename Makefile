.PHONY: build install clean test

BINARY_NAME=cmux
TOGGLE_SCRIPT=cmux-toggle
INSTALL_DIR=$(HOME)/.local/bin

build:
	go build -o bin/$(BINARY_NAME) ./cmd/cmux

install: build
	mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	cp scripts/cmux-toggle.sh $(INSTALL_DIR)/$(TOGGLE_SCRIPT)
	chmod +x $(INSTALL_DIR)/$(TOGGLE_SCRIPT)
	@echo ""
	@echo "Installed to $(INSTALL_DIR)"
	@echo "Add to ~/.tmux.conf:"
	@echo "  bind-key p run-shell \"$(INSTALL_DIR)/$(TOGGLE_SCRIPT)\""

clean:
	rm -rf bin/ $(BINARY_NAME)

test:
	go test ./...
