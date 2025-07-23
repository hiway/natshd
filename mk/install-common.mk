# Common installation targets that delegate to OS-specific implementations

.PHONY: setup install installuser uninstall uninstalluser

# Setup dependencies using OS-appropriate package manager
setup:
	@$(MAKE) $(OS)-setup

# System-wide installation
install:
	@$(MAKE) $(OS)-install

# User installation
installuser:
	@$(MAKE) $(OS)-installuser

# System-wide uninstallation
uninstall:
	@$(MAKE) $(OS)-uninstall

# User uninstallation
uninstalluser:
	@$(MAKE) $(OS)-uninstalluser

# Help target for installation commands
install-help:
	@echo "Available installation targets:"
	@echo "  setup       - Install dependencies using OS package manager"
	@echo "  install     - Install natshd system-wide (requires sudo)"
	@echo "  installuser - Install natshd for current user only"
	@echo "  uninstall   - Remove system-wide installation (requires sudo)"
	@echo "  uninstalluser - Remove user installation"
	@echo ""
	@echo "OS-specific targets are also available:"
	@echo "  linux-setup, linux-install, linux-installuser"
	@echo "  freebsd-setup, freebsd-install, freebsd-installuser"
	@echo "  macos-setup, macos-install, macos-installuser"
	@echo ""
	@echo "Detected OS: $(OS)"
	@echo "Package Manager: $(PKG_MANAGER)"
