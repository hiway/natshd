# Scripts management targets

.PHONY: scripts-install scripts-uninstall scripts-list scripts-backup scripts-update scripts-help

# Install/update scripts only (system-wide)
scripts-install:
	@$(MAKE) $(OS)-scripts-install

# Install/update scripts for user only
scripts-installuser:
	@$(MAKE) $(OS)-scripts-installuser

# Remove scripts (system-wide)
scripts-uninstall:
	@$(MAKE) $(OS)-scripts-uninstall

# Remove scripts (user)
scripts-uninstalluser:
	@$(MAKE) $(OS)-scripts-uninstalluser

# List installed scripts
scripts-list:
	@echo "=== System Scripts ==="
	@if [ -d "$(SYSTEM_SCRIPTS_DIR)" ]; then \
		ls -la $(SYSTEM_SCRIPTS_DIR) 2>/dev/null || echo "No system scripts directory found"; \
	else \
		echo "System scripts directory does not exist: $(SYSTEM_SCRIPTS_DIR)"; \
	fi
	@echo ""
	@echo "=== User Scripts ==="
	@if [ -d "$(USER_SCRIPTS_DIR)" ]; then \
		ls -la $(USER_SCRIPTS_DIR) 2>/dev/null || echo "No user scripts directory found"; \
	else \
		echo "User scripts directory does not exist: $(USER_SCRIPTS_DIR)"; \
	fi

# Backup current scripts
scripts-backup:
	@echo "Backing up scripts..."
	@timestamp=$$(date +%Y%m%d_%H%M%S); \
	if [ -d "$(SYSTEM_SCRIPTS_DIR)" ]; then \
		sudo tar -czf /tmp/natshd-system-scripts-backup-$$timestamp.tar.gz -C $(SYSTEM_SCRIPTS_DIR) . && \
		echo "System scripts backed up to: /tmp/natshd-system-scripts-backup-$$timestamp.tar.gz"; \
	fi; \
	if [ -d "$(USER_SCRIPTS_DIR)" ]; then \
		tar -czf /tmp/natshd-user-scripts-backup-$$timestamp.tar.gz -C $(USER_SCRIPTS_DIR) . && \
		echo "User scripts backed up to: /tmp/natshd-user-scripts-backup-$$timestamp.tar.gz"; \
	fi

# Update scripts (backup first, then install)
scripts-update: scripts-backup scripts-install

# Help for scripts management
scripts-help:
	@echo "Scripts management targets:"
	@echo "  scripts-install      - Install/update scripts system-wide (requires sudo)"
	@echo "  scripts-installuser  - Install/update scripts for current user"
	@echo "  scripts-uninstall    - Remove system scripts (requires sudo)"
	@echo "  scripts-uninstalluser - Remove user scripts"
	@echo "  scripts-list         - List all installed scripts"
	@echo "  scripts-backup       - Backup current scripts to /tmp"
	@echo "  scripts-update       - Backup then update scripts"
	@echo "  scripts-help         - Show this help"
	@echo ""
	@echo "Current paths:"
	@echo "  System scripts: $(SYSTEM_SCRIPTS_DIR)"
	@echo "  User scripts:   $(USER_SCRIPTS_DIR)"
