# FreeBSD-specific setup and installation targets

.PHONY: freebsd-setup freebsd-install freebsd-installuser freebsd-uninstall freebsd-uninstalluser
.PHONY: freebsd-scripts-install freebsd-scripts-installuser freebsd-scripts-uninstall freebsd-scripts-uninstalluser

# Setup dependencies for FreeBSD
freebsd-setup:
	@echo "Setting up dependencies for FreeBSD..."
	sudo pkg update
	sudo pkg install -y jq curl
	@echo "FreeBSD dependencies installed successfully."

# System-wide installation for FreeBSD
freebsd-install: build
	@echo "Installing natshd system-wide on FreeBSD..."
	sudo mkdir -p $(SYSTEM_BIN_DIR)
	sudo mkdir -p $(SYSTEM_CONFIG_DIR)
	sudo mkdir -p $(SYSTEM_SERVICE_DIR)
	sudo mkdir -p $(SYSTEM_SCRIPTS_DIR)
	sudo mkdir -p /var/log
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
	@# Create RC script
	@echo "Creating RC script..."
	@sed -e 's|SYSTEM_BIN_DIR_PLACEHOLDER|$(SYSTEM_BIN_DIR)|g' \
	     -e 's|SYSTEM_CONFIG_DIR_PLACEHOLDER|$(SYSTEM_CONFIG_DIR)|g' \
	     mk/templates/freebsd-system.rc | sudo tee $(SYSTEM_SERVICE_DIR)/natshd > /dev/null
	sudo chmod +x $(SYSTEM_SERVICE_DIR)/natshd
	@# Create natshd user if it doesn't exist
	@if ! id natshd >/dev/null 2>&1; then \
		sudo pw useradd natshd -c "NATS Shell Daemon" -d /var/lib/natshd -s /usr/sbin/nologin; \
		sudo mkdir -p /var/lib/natshd; \
		sudo chown natshd:natshd /var/lib/natshd; \
	fi
	@# Create log file with proper permissions
	@if [ ! -f /var/log/natshd.log ]; then \
		sudo touch /var/log/natshd.log; \
		sudo chown natshd:natshd /var/log/natshd.log; \
		sudo chmod 640 /var/log/natshd.log; \
	fi
	@# Install log rotation configuration
	@echo "Installing log rotation configuration..."
	sudo cp mk/templates/freebsd-newsyslog.conf /etc/newsyslog.conf.d/natshd.conf
	@echo "natshd installed system-wide. Add 'natshd_enable=\"YES\"' to /etc/rc.conf to enable at boot."
	@echo "Use 'sudo service natshd start' to start the service."
	@echo "Logs will be written to /var/log/natshd.log"

# User installation for FreeBSD
freebsd-installuser: build
	@echo "Installing natshd for current user on FreeBSD..."
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
	@# Create user service script
	@echo "Creating user service script..."
	@sed -e 's|USER_BIN_DIR_PLACEHOLDER|$(USER_BIN_DIR)|g' \
	     -e 's|USER_CONFIG_DIR_PLACEHOLDER|$(USER_CONFIG_DIR)|g' \
	     mk/templates/freebsd-user.sh > $(USER_SERVICE_DIR)/natshd.sh
	chmod +x $(USER_SERVICE_DIR)/natshd.sh
	@echo "natshd installed for user. You can start it manually with:"
	@echo "$(USER_SERVICE_DIR)/natshd.sh &"

# System-wide uninstallation for FreeBSD
freebsd-uninstall:
	@echo "Uninstalling natshd system-wide on FreeBSD..."
	sudo service natshd stop 2>/dev/null || true
	sudo rm -f $(SYSTEM_SERVICE_DIR)/natshd
	sudo rm -f $(SYSTEM_BIN_DIR)/$(BINARY_NAME)
	sudo rm -rf $(SYSTEM_CONFIG_DIR)
	sudo rm -rf $(SYSTEM_SCRIPTS_DIR)
	sudo rm -f /var/log/natshd.log
	sudo rm -f /etc/newsyslog.conf.d/natshd.conf
	@echo "natshd uninstalled system-wide. Remove 'natshd_enable=\"YES\"' from /etc/rc.conf if present."

# User uninstallation for FreeBSD
freebsd-uninstalluser:
	@echo "Uninstalling natshd for current user on FreeBSD..."
	pkill -f "$(USER_BIN_DIR)/natshd" 2>/dev/null || true
	rm -f $(USER_SERVICE_DIR)/natshd.sh
	rm -f $(USER_BIN_DIR)/$(BINARY_NAME)
	rm -rf $(USER_CONFIG_DIR)
	rm -rf $(USER_SCRIPTS_DIR)
	@echo "natshd uninstalled for user."

# Scripts-only management for FreeBSD
freebsd-scripts-install:
	@echo "Installing scripts system-wide on FreeBSD..."
	sudo mkdir -p $(SYSTEM_SCRIPTS_DIR)
	sudo cp -r scripts/* $(SYSTEM_SCRIPTS_DIR)/
	sudo find $(SYSTEM_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@echo "System scripts installed to $(SYSTEM_SCRIPTS_DIR)"

freebsd-scripts-installuser:
	@echo "Installing scripts for current user on FreeBSD..."
	mkdir -p $(USER_SCRIPTS_DIR)
	cp -r scripts/* $(USER_SCRIPTS_DIR)/
	find $(USER_SCRIPTS_DIR) -name "*.sh" -exec chmod +x {} \;
	@echo "User scripts installed to $(USER_SCRIPTS_DIR)"

freebsd-scripts-uninstall:
	@echo "Removing system scripts on FreeBSD..."
	sudo rm -rf $(SYSTEM_SCRIPTS_DIR)
	@echo "System scripts removed."

freebsd-scripts-uninstalluser:
	@echo "Removing user scripts on FreeBSD..."
	rm -rf $(USER_SCRIPTS_DIR)
	@echo "User scripts removed."
