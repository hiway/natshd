# Makefile Modules

This directory contains modularized Makefile fragments that provide OS-specific installation and setup targets for natshd.

## Structure

### Core Files

- `os-detect.mk` - OS detection and variable setup
- `install-common.mk` - Common installation targets that delegate to OS-specific implementations

### OS-Specific Installation Files

- `linux-install.mk` - Linux-specific setup, install, and uninstall targets
- `freebsd-install.mk` - FreeBSD-specific setup, install, and uninstall targets  
- `macos-install.mk` - macOS-specific setup, install, and uninstall targets

### Templates Directory

Contains service definition templates that are processed by the installation targets:

- `linux-system.service` - systemd system service template for Linux
- `linux-user.service` - systemd user service template for Linux
- `freebsd-system.rc` - RC script template for FreeBSD system installation
- `freebsd-user.sh` - User service script template for FreeBSD
- `macos-system.plist` - LaunchDaemon plist template for macOS system installation
- `macos-user.plist` - LaunchAgent plist template for macOS user installation

## Available Targets

### Common Targets (Auto-detect OS)

- `setup` - Install dependencies using OS package manager
- `install` - Install natshd system-wide (requires sudo)
- `installuser` - Install natshd for current user only
- `uninstall` - Remove system-wide installation (requires sudo)
- `uninstalluser` - Remove user installation

### OS-Specific Targets

Each OS has its own set of targets:

- `{os}-setup` - Install dependencies
- `{os}-install` - System-wide installation
- `{os}-installuser` - User installation
- `{os}-uninstall` - System-wide removal
- `{os}-uninstalluser` - User removal

Where `{os}` is one of: `linux`, `freebsd`, `macos`

### Utility Targets

- `debug-os` - Show detected OS and installation paths
- `install-help` - Show available installation targets

## Template System

Templates use placeholder strings that are replaced during installation:

- `SYSTEM_BIN_DIR_PLACEHOLDER` → System binary directory
- `SYSTEM_CONFIG_DIR_PLACEHOLDER` → System configuration directory
- `USER_BIN_DIR_PLACEHOLDER` → User binary directory
- `USER_CONFIG_DIR_PLACEHOLDER` → User configuration directory
- `HOME_PLACEHOLDER` → User home directory

## OS-Specific Behavior

### Linux

- Uses systemd for service management
- Supports multiple package managers (apt, yum, dnf, pacman, apk)
- Creates dedicated `natshd` user for system installation
- Installs to `/usr/local/bin` (system) or `~/.local/bin` (user)

### FreeBSD

- Uses RC scripts for service management
- Uses `pkg` package manager
- Creates dedicated `natshd` user for system installation
- Installs to `/usr/local/bin` (system) or `~/bin` (user)

### macOS

- Uses LaunchDaemons/LaunchAgents for service management
- Supports Homebrew and MacPorts package managers
- Creates `_natshd` user for system installation
- Installs to `/usr/local/bin` (system) or `~/.local/bin` (user)

## Usage

Include all modules in your main Makefile:

```makefile
include mk/os-detect.mk
include mk/install-common.mk
include mk/linux-install.mk
include mk/freebsd-install.mk
include mk/macos-install.mk
```

Then use the common targets which will automatically delegate to the appropriate OS-specific implementation.
