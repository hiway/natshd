# macOS-specific setup and installation targets

.PHONY: macos-setup macos-install macos-installuser macos-uninstall macos-uninstalluser
.PHONY: macos-scripts-install macos-scripts-installuser macos-scripts-uninstall macos-scripts-uninstalluser

# Setup dependencies for macOS
macos-setup:
	@echo "Setting up dependencies for macOS..."
	@if command -v brew >/dev/null 2>&1; then \
		brew update && brew install jq curl; \
	elif command -v port >/dev/null 2>&1; then \
		sudo port selfupdate && sudo port install jq curl; \
	else \
		echo "No supported package manager found. Please install Homebrew or MacPorts first."; \
		echo "Homebrew: /bin/bash -c \"\$$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""; \
		echo "MacPorts: https://www.macports.org/install.php"; \
		exit 1; \
	fi
	@echo "macOS dependencies installed successfully."

# System-wide installation for macOS
macos-install: build
	@echo "Installing natshd system-wide on macOS..."
	sudo mkdir -p $(SYSTEM_BIN_DIR)
	sudo mkdir -p $(SYSTEM_CONFIG_DIR)
	sudo mkdir -p $(SYSTEM_SERVICE_DIR)
	sudo mkdir -p $(SYSTEM_SCRIPTS_DIR)
	sudo cp $(BINARY_NAME) $(SYSTEM_BIN_DIR)/
	sudo chmod +x $(SYSTEM_BIN_DIR)/$(BINARY_NAME)
	@# Install scripts
	@echo "Installing scripts..."
	sudo cp -r scripts/* $(SYSTEM_SCRIPTS_DIR)/
	sudo find $(SYSTEM_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@# Install configuration
	@echo "Installing configuration..."
	@sed 's|SYSTEM_SCRIPTS_DIR_PLACEHOLDER|$(SYSTEM_SCRIPTS_DIR)|g' \
	     mk/templates/system-config.toml | sudo tee $(SYSTEM_CONFIG_DIR)/config.toml > /dev/null
	@# Create LaunchDaemon plist
	@echo "Creating LaunchDaemon plist..."
	@sed 's|SYSTEM_BIN_DIR_PLACEHOLDER|$(SYSTEM_BIN_DIR)|g; s|SYSTEM_CONFIG_DIR_PLACEHOLDER|$(SYSTEM_CONFIG_DIR)|g' mk/templates/macos-system.plist | sudo tee $(SYSTEM_SERVICE_DIR)/io.nats.natshd.plist > /dev/null
	@# Create natshd user if it doesn't exist
	@if ! dscl . -read /Users/_natshd >/dev/null 2>&1; then \
		echo "Creating _natshd user..."; \
		sudo dscl . -create /Users/_natshd; \
		sudo dscl . -create /Users/_natshd UserShell /usr/bin/false; \
		sudo dscl . -create /Users/_natshd RealName "NATS Shell Daemon"; \
		sudo dscl . -create /Users/_natshd UniqueID 299; \
		sudo dscl . -create /Users/_natshd PrimaryGroupID 299; \
		sudo dscl . -create /Users/_natshd NFSHomeDirectory /var/lib/natshd; \
		sudo dscl . -create /Groups/_natshd; \
		sudo dscl . -create /Groups/_natshd RealName "NATS Shell Daemon"; \
		sudo dscl . -create /Groups/_natshd PrimaryGroupID 299; \
		sudo mkdir -p /var/lib/natshd; \
		sudo chown _natshd:_natshd /var/lib/natshd; \
	fi
	@echo "natshd installed system-wide. Use 'sudo launchctl load $(SYSTEM_SERVICE_DIR)/io.nats.natshd.plist' to start."

# User installation for macOS
macos-installuser: build
	@echo "Installing natshd for current user on macOS..."
	mkdir -p $(USER_BIN_DIR)
	mkdir -p $(USER_CONFIG_DIR)
	mkdir -p $(USER_SERVICE_DIR)
	mkdir -p $(USER_SCRIPTS_DIR)
	cp $(BINARY_NAME) $(USER_BIN_DIR)/
	chmod +x $(USER_BIN_DIR)/$(BINARY_NAME)
	@# Install scripts
	@echo "Installing scripts..."
	cp -r scripts/* $(USER_SCRIPTS_DIR)/
	find $(USER_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@# Install configuration
	@echo "Installing configuration..."
	@sed 's|USER_SCRIPTS_DIR_PLACEHOLDER|$(USER_SCRIPTS_DIR)|g' \
	     mk/templates/user-config.toml > $(USER_CONFIG_DIR)/config.toml
	@# Create user LaunchAgent plist
	@echo "Creating user LaunchAgent plist..."
	@sed 's|USER_BIN_DIR_PLACEHOLDER|$(USER_BIN_DIR)|g; s|USER_CONFIG_DIR_PLACEHOLDER|$(USER_CONFIG_DIR)|g; s|HOME_PLACEHOLDER|$(HOME)|g' mk/templates/macos-user.plist > $(USER_SERVICE_DIR)/io.nats.natshd.plist
	mkdir -p $(HOME)/Library/Logs
	@echo "natshd installed for user. Use 'launchctl load $(USER_SERVICE_DIR)/io.nats.natshd.plist' to start."

# System-wide uninstallation for macOS
macos-uninstall:
	@echo "Uninstalling natshd system-wide on macOS..."
	sudo launchctl unload $(SYSTEM_SERVICE_DIR)/io.nats.natshd.plist 2>/dev/null || true
	sudo rm -f $(SYSTEM_SERVICE_DIR)/io.nats.natshd.plist
	sudo rm -f $(SYSTEM_BIN_DIR)/$(BINARY_NAME)
	sudo rm -rf $(SYSTEM_CONFIG_DIR)
	sudo rm -rf $(SYSTEM_SCRIPTS_DIR)
	@echo "natshd uninstalled system-wide."

# User uninstallation for macOS
macos-uninstalluser:
	@echo "Uninstalling natshd for current user on macOS..."
	launchctl unload $(USER_SERVICE_DIR)/io.nats.natshd.plist 2>/dev/null || true
	rm -f $(USER_SERVICE_DIR)/io.nats.natshd.plist
	rm -f $(USER_BIN_DIR)/$(BINARY_NAME)
	rm -rf $(USER_CONFIG_DIR)
	rm -rf $(USER_SCRIPTS_DIR)
	@echo "natshd uninstalled for user."

# Scripts-only management for macOS
macos-scripts-install:
	@echo "Installing scripts system-wide on macOS..."
	sudo mkdir -p $(SYSTEM_SCRIPTS_DIR)
	sudo cp -r scripts/* $(SYSTEM_SCRIPTS_DIR)/
	sudo find $(SYSTEM_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@echo "System scripts installed to $(SYSTEM_SCRIPTS_DIR)"

macos-scripts-installuser:
	@echo "Installing scripts for current user on macOS..."
	mkdir -p $(USER_SCRIPTS_DIR)
	cp -r scripts/* $(USER_SCRIPTS_DIR)/
	find $(USER_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@echo "User scripts installed to $(USER_SCRIPTS_DIR)"

macos-scripts-uninstall:
	@echo "Removing system scripts on macOS..."
	sudo rm -rf $(SYSTEM_SCRIPTS_DIR)
	@echo "System scripts removed."

macos-scripts-uninstalluser:
	@echo "Removing user scripts on macOS..."
	rm -rf $(USER_SCRIPTS_DIR)
	@echo "User scripts removed."
