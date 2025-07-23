# Linux-specific setup and installation targets

.PHONY: linux-setup linux-install linux-installuser linux-uninstall linux-uninstalluser
.PHONY: linux-scripts-install linux-scripts-installuser linux-scripts-uninstall linux-scripts-uninstalluser

# Setup dependencies for Linux
linux-setup:
	@echo "Setting up dependencies for Linux..."
	@if command -v apt-get >/dev/null 2>&1; then \
		sudo apt-get update && sudo apt-get install -y jq curl; \
	elif command -v yum >/dev/null 2>&1; then \
		sudo yum install -y jq curl; \
	elif command -v dnf >/dev/null 2>&1; then \
		sudo dnf install -y jq curl; \
	elif command -v pacman >/dev/null 2>&1; then \
		sudo pacman -S --noconfirm jq curl; \
	elif command -v apk >/dev/null 2>&1; then \
		sudo apk add --no-cache jq curl; \
	else \
		echo "No supported package manager found. Please install jq and curl manually."; \
		exit 1; \
	fi
	@echo "Linux dependencies installed successfully."

# System-wide installation for Linux
linux-install: build
	@echo "Installing natshd system-wide on Linux..."
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
	@if [ -f $(SYSTEM_CONFIG_DIR)/config.toml ]; then \
		echo "Configuration file $(SYSTEM_CONFIG_DIR)/config.toml already exists."; \
		echo -n "Overwrite it? [y/N]: "; \
		read -r response; \
		if [ "$$response" = "y" ] || [ "$$response" = "Y" ]; then \
			sed 's|SYSTEM_SCRIPTS_DIR_PLACEHOLDER|$(SYSTEM_SCRIPTS_DIR)|g' \
			    mk/templates/system-config.toml | sudo tee $(SYSTEM_CONFIG_DIR)/config.toml > /dev/null; \
			echo "Configuration file updated."; \
		else \
			echo "Keeping existing configuration file."; \
		fi; \
	else \
		sed 's|SYSTEM_SCRIPTS_DIR_PLACEHOLDER|$(SYSTEM_SCRIPTS_DIR)|g' \
		    mk/templates/system-config.toml | sudo tee $(SYSTEM_CONFIG_DIR)/config.toml > /dev/null; \
		echo "Configuration file created."; \
	fi
	@# Create systemd service file
	@echo "Creating systemd service file..."
	@sed -e 's|SYSTEM_BIN_DIR_PLACEHOLDER|$(SYSTEM_BIN_DIR)|g' \
	     -e 's|SYSTEM_CONFIG_DIR_PLACEHOLDER|$(SYSTEM_CONFIG_DIR)|g' \
	     mk/templates/linux-system.service | sudo tee $(SYSTEM_SERVICE_DIR)/natshd.service > /dev/null
	@# Create natshd user if it doesn't exist
	@if ! id natshd >/dev/null 2>&1; then \
		sudo useradd -r -s /bin/false -d /var/lib/natshd natshd; \
		sudo mkdir -p /var/lib/natshd; \
		sudo chown natshd:natshd /var/lib/natshd; \
	fi
	sudo systemctl daemon-reload
	@echo "natshd installed system-wide. Use 'sudo systemctl enable natshd' to enable at boot."
	@echo "Use 'sudo systemctl start natshd' to start the service."

# User installation for Linux
linux-installuser: build
	@echo "Installing natshd for current user on Linux..."
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
	@if [ -f $(USER_CONFIG_DIR)/config.toml ]; then \
		echo "Configuration file $(USER_CONFIG_DIR)/config.toml already exists."; \
		echo -n "Overwrite it? [y/N]: "; \
		read -r response; \
		if [ "$$response" = "y" ] || [ "$$response" = "Y" ]; then \
			sed 's|USER_SCRIPTS_DIR_PLACEHOLDER|$(USER_SCRIPTS_DIR)|g' \
			    mk/templates/user-config.toml > $(USER_CONFIG_DIR)/config.toml; \
			echo "Configuration file updated."; \
		else \
			echo "Keeping existing configuration file."; \
		fi; \
	else \
		sed 's|USER_SCRIPTS_DIR_PLACEHOLDER|$(USER_SCRIPTS_DIR)|g' \
		    mk/templates/user-config.toml > $(USER_CONFIG_DIR)/config.toml; \
		echo "Configuration file created."; \
	fi
	@# Create user systemd service file
	@echo "Creating user systemd service file..."
	@sed -e 's|USER_BIN_DIR_PLACEHOLDER|$(USER_BIN_DIR)|g' \
	     -e 's|USER_CONFIG_DIR_PLACEHOLDER|$(USER_CONFIG_DIR)|g' \
	     mk/templates/linux-user.service > $(USER_SERVICE_DIR)/natshd.service
	systemctl --user daemon-reload
	@echo "natshd installed for user. Use 'systemctl --user enable natshd' to enable at login."
	@echo "Use 'systemctl --user start natshd' to start the service."

# System-wide uninstallation for Linux
linux-uninstall:
	@echo "Uninstalling natshd system-wide on Linux..."
	sudo systemctl stop natshd 2>/dev/null || true
	sudo systemctl disable natshd 2>/dev/null || true
	sudo rm -f $(SYSTEM_SERVICE_DIR)/natshd.service
	sudo rm -f $(SYSTEM_BIN_DIR)/$(BINARY_NAME)
	sudo rm -rf $(SYSTEM_CONFIG_DIR)
	sudo rm -rf $(SYSTEM_SCRIPTS_DIR)
	sudo systemctl daemon-reload
	@echo "natshd uninstalled system-wide."

# User uninstallation for Linux
linux-uninstalluser:
	@echo "Uninstalling natshd for current user on Linux..."
	systemctl --user stop natshd 2>/dev/null || true
	systemctl --user disable natshd 2>/dev/null || true
	rm -f $(USER_SERVICE_DIR)/natshd.service
	rm -f $(USER_BIN_DIR)/$(BINARY_NAME)
	rm -rf $(USER_CONFIG_DIR)
	rm -rf $(USER_SCRIPTS_DIR)
	systemctl --user daemon-reload
	@echo "natshd uninstalled for user."

# Scripts-only management for Linux
linux-scripts-install:
	@echo "Installing scripts system-wide on Linux..."
	sudo mkdir -p $(SYSTEM_SCRIPTS_DIR)
	sudo cp -r scripts/* $(SYSTEM_SCRIPTS_DIR)/
	sudo find $(SYSTEM_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@echo "System scripts installed to $(SYSTEM_SCRIPTS_DIR)"

linux-scripts-installuser:
	@echo "Installing scripts for current user on Linux..."
	mkdir -p $(USER_SCRIPTS_DIR)
	cp -r scripts/* $(USER_SCRIPTS_DIR)/
	find $(USER_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@echo "User scripts installed to $(USER_SCRIPTS_DIR)"

linux-scripts-uninstall:
	@echo "Removing system scripts on Linux..."
	sudo rm -rf $(SYSTEM_SCRIPTS_DIR)
	@echo "System scripts removed."

linux-scripts-uninstalluser:
	@echo "Removing user scripts on Linux..."
	rm -rf $(USER_SCRIPTS_DIR)
	@echo "User scripts removed."
