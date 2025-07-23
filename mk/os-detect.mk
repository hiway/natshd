# OS Detection
# This file detects the operating system and sets appropriate variables

# Detect OS
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Set OS-specific variables
ifeq ($(UNAME_S),Linux)
    OS := linux
    PKG_MANAGER := $(shell which apt-get yum dnf pacman brew 2>/dev/null | head -n1)
    SYSTEM_BIN_DIR := /usr/local/bin
    SYSTEM_CONFIG_DIR := /etc/natshd
    SYSTEM_SERVICE_DIR := /etc/systemd/system
    SYSTEM_SCRIPTS_DIR := /usr/local/share/natshd/scripts
    USER_BIN_DIR := $(HOME)/.local/bin
    USER_CONFIG_DIR := $(HOME)/.config/natshd
    USER_SERVICE_DIR := $(HOME)/.config/systemd/user
    USER_SCRIPTS_DIR := $(HOME)/.local/share/natshd/scripts
endif

ifeq ($(UNAME_S),FreeBSD)
    OS := freebsd
    PKG_MANAGER := pkg
    SYSTEM_BIN_DIR := /usr/local/bin
    SYSTEM_CONFIG_DIR := /usr/local/etc/natshd
    SYSTEM_SERVICE_DIR := /usr/local/etc/rc.d
    SYSTEM_SCRIPTS_DIR := /usr/local/share/natshd/scripts
    USER_BIN_DIR := $(HOME)/bin
    USER_CONFIG_DIR := $(HOME)/.config/natshd
    USER_SERVICE_DIR := $(HOME)/.config/service
    USER_SCRIPTS_DIR := $(HOME)/.local/share/natshd/scripts
endif

ifeq ($(UNAME_S),Darwin)
    OS := macos
    PKG_MANAGER := $(shell which brew port 2>/dev/null | head -n1)
    SYSTEM_BIN_DIR := /usr/local/bin
    SYSTEM_CONFIG_DIR := /usr/local/etc/natshd
    SYSTEM_SERVICE_DIR := /Library/LaunchDaemons
    SYSTEM_SCRIPTS_DIR := /usr/local/share/natshd/scripts
    USER_BIN_DIR := $(HOME)/.local/bin
    USER_CONFIG_DIR := $(HOME)/.config/natshd
    USER_SERVICE_DIR := $(HOME)/Library/LaunchAgents
    USER_SCRIPTS_DIR := $(HOME)/Library/Application Support/natshd/scripts
endif

# Export variables for use in other makefiles
export OS PKG_MANAGER SYSTEM_BIN_DIR SYSTEM_CONFIG_DIR SYSTEM_SERVICE_DIR SYSTEM_SCRIPTS_DIR
export USER_BIN_DIR USER_CONFIG_DIR USER_SERVICE_DIR USER_SCRIPTS_DIR UNAME_S UNAME_M

# Debug target to show detected values
debug-os:
	@echo "Detected OS: $(OS)"
	@echo "Package Manager: $(PKG_MANAGER)"
	@echo "System Bin Dir: $(SYSTEM_BIN_DIR)"
	@echo "System Config Dir: $(SYSTEM_CONFIG_DIR)"
	@echo "System Service Dir: $(SYSTEM_SERVICE_DIR)"
	@echo "System Scripts Dir: $(SYSTEM_SCRIPTS_DIR)"
	@echo "User Bin Dir: $(USER_BIN_DIR)"
	@echo "User Config Dir: $(USER_CONFIG_DIR)"
	@echo "User Service Dir: $(USER_SERVICE_DIR)"
	@echo "User Scripts Dir: $(USER_SCRIPTS_DIR)"
