# AI Work Studio Installation Guide

**Simplicity through experience, not complexity through programming**

This guide covers installation of AI Work Studio on all supported platforms. AI Work Studio is a goal-directed autonomous agent system that serves as your personal AI assistant, learning and adapting to your specific needs over time.

## Table of Contents

- [System Requirements](#system-requirements)
- [Quick Install (Automated)](#quick-install-automated)
- [Manual Installation](#manual-installation)
- [Platform-Specific Notes](#platform-specific-notes)
- [Configuration](#configuration)
- [Service Setup](#service-setup)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)
- [Uninstallation](#uninstallation)

## System Requirements

### Minimum Requirements
- **Operating System**: Linux (Ubuntu 18.04+), macOS (10.15+), or Windows 10+
- **Go**: Version 1.21.0 or later
- **Git**: For downloading source code
- **Memory**: 512 MB RAM minimum, 1 GB recommended
- **Storage**: 100 MB for binaries, additional space for data and models

### Recommended
- **Memory**: 2 GB RAM or more for better performance
- **Storage**: 1 GB+ free space for local models and data
- **Network**: Internet connection for remote LLM providers and updates

### Dependencies
- **Go 1.21+**: Download from [golang.org](https://golang.org/dl/)
- **Git**: Install via your system package manager
- **Make** (optional): For easier building, available on most Unix systems

## Quick Install (Automated)

### Unix/Linux/macOS

```bash
# Download and run the installer
curl -sSL https://raw.githubusercontent.com/yourusername/ai-work-studio/main/install.sh | bash

# Or clone and run locally
git clone https://github.com/yourusername/ai-work-studio.git
cd ai-work-studio
chmod +x install.sh
./install.sh
```

The installer will:
- Check system requirements
- Download dependencies and build binaries
- Set up data directories
- Configure the system interactively
- Optionally install systemd service

### Windows (PowerShell)

```powershell
# Download and run the installer (run as Administrator for system-wide install)
iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/yourusername/ai-work-studio/main/install.ps1'))

# Or clone and run locally
git clone https://github.com/yourusername/ai-work-studio.git
cd ai-work-studio
.\install.ps1
```

**PowerShell Options:**
```powershell
.\install.ps1 -NoInteractive                      # Silent installation
.\install.ps1 -InstallDir "C:\Tools\AIStudio"    # Custom install directory
.\install.ps1 -Help                               # Show help
```

## Manual Installation

If you prefer to install manually or the automated installer doesn't work for your setup:

### Step 1: Install Dependencies

**Go Installation:**
```bash
# Check if Go is installed
go version

# If not installed, download from https://golang.org/dl/
# Linux example:
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile
```

**Git Installation:**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install git

# CentOS/RHEL/Fedora
sudo yum install git  # or dnf install git

# macOS
xcode-select --install  # or brew install git

# Windows
# Download from https://git-scm.com/download/win
```

### Step 2: Clone and Build

```bash
# Clone the repository
git clone https://github.com/yourusername/ai-work-studio.git
cd ai-work-studio

# Build using Make (recommended)
make build

# Or build manually
go mod download
go mod tidy
mkdir -p bin
go build -o bin/ai-work-studio ./cmd/studio
go build -o bin/ai-work-studio-agent ./cmd/agent
```

### Step 3: Install Binaries

**Unix/Linux/macOS:**
```bash
# User installation (recommended)
mkdir -p ~/.local/bin
cp bin/ai-work-studio ~/.local/bin/
cp bin/ai-work-studio-agent ~/.local/bin/
chmod +x ~/.local/bin/ai-work-studio*

# Add to PATH if not already
echo 'export PATH=$HOME/.local/bin:$PATH' >> ~/.bashrc
source ~/.bashrc

# System-wide installation (requires sudo)
sudo cp bin/ai-work-studio /usr/local/bin/
sudo cp bin/ai-work-studio-agent /usr/local/bin/
sudo chmod +x /usr/local/bin/ai-work-studio*
```

**Windows:**
```powershell
# Create installation directory
New-Item -ItemType Directory -Path "$env:LOCALAPPDATA\Programs\AIWorkStudio" -Force

# Copy binaries
Copy-Item "bin\ai-work-studio.exe" "$env:LOCALAPPDATA\Programs\AIWorkStudio\"
Copy-Item "bin\ai-work-studio-agent.exe" "$env:LOCALAPPDATA\Programs\AIWorkStudio\"

# Add to PATH (requires restart or new shell)
$newPath = [Environment]::GetEnvironmentVariable("PATH", "User") + ";$env:LOCALAPPDATA\Programs\AIWorkStudio"
[Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
```

### Step 4: Set Up Data Directories

**Unix/Linux/macOS:**
```bash
mkdir -p ~/.config/ai-work-studio/{nodes,edges,backups,cache,methods,keys}
mkdir -p ~/.local/share/ai-work-studio/logs
```

**Windows:**
```powershell
New-Item -ItemType Directory -Path "$env:APPDATA\AIWorkStudio\nodes" -Force
New-Item -ItemType Directory -Path "$env:APPDATA\AIWorkStudio\edges" -Force
New-Item -ItemType Directory -Path "$env:APPDATA\AIWorkStudio\backups" -Force
New-Item -ItemType Directory -Path "$env:APPDATA\AIWorkStudio\cache" -Force
New-Item -ItemType Directory -Path "$env:APPDATA\AIWorkStudio\methods" -Force
New-Item -ItemType Directory -Path "$env:APPDATA\AIWorkStudio\keys" -Force
New-Item -ItemType Directory -Path "$env:LOCALAPPDATA\AIWorkStudio\logs" -Force
```

### Step 5: Create Configuration

Create a configuration file at the appropriate location:

**Unix/Linux/macOS:** `~/.config/ai-work-studio/config.json`
**Windows:** `%APPDATA%\AIWorkStudio\config.json`

```json
{
  "version": "1.0",
  "data_directory": "~/.config/ai-work-studio",
  "log_directory": "~/.local/share/ai-work-studio/logs",
  "log_level": "info",
  "storage": {
    "type": "file",
    "backup_enabled": true,
    "backup_interval": "24h"
  },
  "llm": {
    "default_provider": "local",
    "budget": {
      "daily_limit": 100.0,
      "warn_threshold": 80.0
    }
  },
  "agent": {
    "auto_start": false,
    "check_interval": "5m"
  }
}
```

**Windows Configuration Example:**
```json
{
  "version": "1.0",
  "data_directory": "%APPDATA%\\AIWorkStudio",
  "log_directory": "%LOCALAPPDATA%\\AIWorkStudio\\logs",
  "log_level": "info",
  "storage": {
    "type": "file",
    "backup_enabled": true,
    "backup_interval": "24h"
  },
  "llm": {
    "default_provider": "local",
    "budget": {
      "daily_limit": 100.0,
      "warn_threshold": 80.0
    }
  },
  "agent": {
    "auto_start": false,
    "check_interval": "5m"
  }
}
```

## Platform-Specific Notes

### Linux

**Package Manager Installation:**
We're working on packages for major distributions. For now, use the automated installer or manual build.

**Dependencies on Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install build-essential git curl
```

**Dependencies on CentOS/RHEL/Fedora:**
```bash
sudo yum groupinstall "Development Tools"  # or dnf
sudo yum install git curl
```

**SELinux considerations:**
If you encounter permission issues on SELinux-enabled systems:
```bash
sudo setsebool -P httpd_can_network_connect 1
sudo restorecon -R /usr/local/bin/ai-work-studio*
```

### macOS

**Using Homebrew** (when available):
```bash
# Coming soon
brew tap yourusername/ai-work-studio
brew install ai-work-studio
```

**Gatekeeper and Notarization:**
The binaries are not yet notarized. If you get security warnings:
1. Try to run the binary
2. Go to System Preferences > Security & Privacy
3. Click "Allow" for the blocked application

**Apple Silicon (M1/M2) Notes:**
The software runs natively on Apple Silicon. Use the `darwin-arm64` release or build from source.

### Windows

**PowerShell Execution Policy:**
You may need to adjust the execution policy to run the installer:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Windows Defender:**
Windows Defender may flag the binaries as potentially unwanted software (PUA) since they're not signed. Add exclusions if needed:
1. Open Windows Security
2. Go to Virus & threat protection
3. Add exclusions for the installation directory

**Windows Service:**
The installer can set up a Windows service for the background agent. This requires administrator privileges.

**WSL (Windows Subsystem for Linux):**
AI Work Studio works in WSL. Use the Linux installation instructions within your WSL environment.

## Configuration

### Initial Setup

On first run, AI Work Studio will guide you through initial configuration:

```bash
ai-work-studio --setup
```

This will prompt for:
- Data storage location
- LLM provider preferences
- API keys (if using remote providers)
- Budget limits
- Agent settings

### LLM Providers

AI Work Studio supports multiple LLM providers:

**Local Models:**
- No API keys required
- Privacy-focused
- No per-request costs
- Requires more local resources

**Anthropic Claude API:**
1. Get API key from [console.anthropic.com](https://console.anthropic.com/)
2. Store in `config/keys/anthropic.key` or use environment variable `ANTHROPIC_API_KEY`

**OpenAI API:**
1. Get API key from [platform.openai.com](https://platform.openai.com/api-keys)
2. Store in `config/keys/openai.key` or use environment variable `OPENAI_API_KEY`

### Budget Management

Set spending limits to control API costs:

```json
{
  "llm": {
    "budget": {
      "daily_limit": 50.0,
      "monthly_limit": 1000.0,
      "warn_threshold": 80.0,
      "currency": "USD"
    }
  }
}
```

## Service Setup

### Linux (systemd)

**System Service:**
```bash
# Copy service file
sudo cp systemd/ai-work-studio.service /etc/systemd/system/

# Create dedicated user
sudo useradd --system --shell /bin/false --home-dir /var/lib/ai-work-studio \
  --create-home ai-work-studio

# Set up directories
sudo mkdir -p /var/lib/ai-work-studio/{data,logs}
sudo mkdir -p /etc/ai-work-studio
sudo chown -R ai-work-studio:ai-work-studio /var/lib/ai-work-studio

# Copy and adjust configuration
sudo cp ~/.config/ai-work-studio/config.json /etc/ai-work-studio/
sudo chown ai-work-studio:ai-work-studio /etc/ai-work-studio/config.json

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable ai-work-studio
sudo systemctl start ai-work-studio
```

**User Service:**
```bash
# Copy user service file
mkdir -p ~/.config/systemd/user
cp systemd/ai-work-studio-user.service ~/.config/systemd/user/ai-work-studio.service

# Enable and start
systemctl --user daemon-reload
systemctl --user enable ai-work-studio
systemctl --user start ai-work-studio

# Auto-start on login
sudo loginctl enable-linger $USER
```

**Service Management:**
```bash
# Check status
systemctl status ai-work-studio
# or
systemctl --user status ai-work-studio

# View logs
journalctl -u ai-work-studio -f
# or
journalctl --user -u ai-work-studio -f

# Restart
sudo systemctl restart ai-work-studio
# or
systemctl --user restart ai-work-studio
```

### macOS (launchd)

Create a launch agent plist file at `~/Library/LaunchAgents/com.yourusername.ai-work-studio.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.yourusername.ai-work-studio</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/username/.local/bin/ai-work-studio-agent</string>
        <string>--config</string>
        <string>/Users/username/.config/ai-work-studio/config.json</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/username/.local/share/ai-work-studio/logs/agent.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/username/.local/share/ai-work-studio/logs/agent-error.log</string>
</dict>
</plist>
```

Load and start:
```bash
launchctl load ~/Library/LaunchAgents/com.yourusername.ai-work-studio.plist
launchctl start com.yourusername.ai-work-studio
```

### Windows Service

The PowerShell installer can set up a Windows service. Manual setup:

```powershell
# Run as Administrator
sc create "AIWorkStudioAgent" binPath= '"C:\Users\username\AppData\Local\Programs\AIWorkStudio\ai-work-studio-agent.exe" --config "C:\Users\username\AppData\Roaming\AIWorkStudio\config.json"' start= auto
sc description "AIWorkStudioAgent" "AI Work Studio background agent service"
sc start "AIWorkStudioAgent"
```

## Verification

### Test Installation

```bash
# Check binary versions
ai-work-studio --version
ai-work-studio-agent --version

# Test basic functionality
ai-work-studio --help
ai-work-studio --config-check

# Run agent in foreground (for testing)
ai-work-studio-agent --config ~/.config/ai-work-studio/config.json --debug
```

### Verify Data Directories

```bash
# Unix/Linux/macOS
ls -la ~/.config/ai-work-studio/
ls -la ~/.local/share/ai-work-studio/logs/

# Windows
dir "%APPDATA%\AIWorkStudio"
dir "%LOCALAPPDATA%\AIWorkStudio\logs"
```

### Test GUI

```bash
# Launch the GUI application
ai-work-studio

# Should open a window with the AI Work Studio interface
```

### Health Check

```bash
# Built-in health check
ai-work-studio-agent --health-check

# Check service status
systemctl status ai-work-studio  # Linux
launchctl list | grep ai-work-studio  # macOS
sc query "AIWorkStudioAgent"  # Windows
```

## Troubleshooting

### Common Issues

**"Go not found":**
- Ensure Go 1.21+ is installed and in your PATH
- Restart your shell after installation

**"Permission denied":**
- Ensure binaries have execute permissions: `chmod +x /path/to/binary`
- Check directory permissions for data folders

**"Port already in use":**
- Another instance may be running
- Check with: `ps aux | grep ai-work-studio`
- Kill existing processes or change port in config

**GUI doesn't start:**
- Ensure you have a display server running
- On Linux, check `$DISPLAY` environment variable
- Try running from terminal to see error messages

**Service won't start:**
- Check service logs: `journalctl -u ai-work-studio`
- Verify configuration file syntax: `ai-work-studio --config-check`
- Ensure all paths in config exist and are accessible

### Log Locations

**Linux:**
- Journal: `journalctl -u ai-work-studio`
- Files: `~/.local/share/ai-work-studio/logs/`

**macOS:**
- Console app or: `log show --predicate 'process == "ai-work-studio-agent"'`
- Files: `~/.local/share/ai-work-studio/logs/`

**Windows:**
- Event Viewer > Windows Logs > Application
- Files: `%LOCALAPPDATA%\AIWorkStudio\logs\`

### Debug Mode

Run with debug logging:
```bash
ai-work-studio --debug
ai-work-studio-agent --debug --config /path/to/config.json
```

### Network Issues

If you have connection problems:
```bash
# Test API connectivity
curl -I https://api.anthropic.com/v1/health
curl -I https://api.openai.com/v1/models

# Check proxy settings
echo $HTTP_PROXY $HTTPS_PROXY

# Test DNS resolution
nslookup api.anthropic.com
```

### Reset Configuration

To start fresh:
```bash
# Backup existing data
cp -r ~/.config/ai-work-studio ~/.config/ai-work-studio.backup

# Remove configuration
rm -rf ~/.config/ai-work-studio
rm -rf ~/.local/share/ai-work-studio

# Re-run setup
ai-work-studio --setup
```

## Uninstallation

### Remove Binaries

**Unix/Linux/macOS:**
```bash
# User installation
rm ~/.local/bin/ai-work-studio*

# System installation
sudo rm /usr/local/bin/ai-work-studio*
```

**Windows:**
```powershell
Remove-Item "$env:LOCALAPPDATA\Programs\AIWorkStudio" -Recurse -Force
# Remove from PATH manually through System Properties > Environment Variables
```

### Remove Data

**Warning:** This will delete all your data, goals, and learned methods.

**Unix/Linux/macOS:**
```bash
rm -rf ~/.config/ai-work-studio
rm -rf ~/.local/share/ai-work-studio
```

**Windows:**
```powershell
Remove-Item "$env:APPDATA\AIWorkStudio" -Recurse -Force
Remove-Item "$env:LOCALAPPDATA\AIWorkStudio" -Recurse -Force
```

### Remove Services

**Linux:**
```bash
sudo systemctl stop ai-work-studio
sudo systemctl disable ai-work-studio
sudo rm /etc/systemd/system/ai-work-studio.service
sudo systemctl daemon-reload

# Or user service
systemctl --user stop ai-work-studio
systemctl --user disable ai-work-studio
rm ~/.config/systemd/user/ai-work-studio.service
systemctl --user daemon-reload
```

**macOS:**
```bash
launchctl unload ~/Library/LaunchAgents/com.yourusername.ai-work-studio.plist
rm ~/Library/LaunchAgents/com.yourusername.ai-work-studio.plist
```

**Windows:**
```powershell
sc stop "AIWorkStudioAgent"
sc delete "AIWorkStudioAgent"
```

## Support

- **Documentation**: [GitHub Repository](https://github.com/yourusername/ai-work-studio)
- **Issues**: [GitHub Issues](https://github.com/yourusername/ai-work-studio/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/ai-work-studio/discussions)

## License

AI Work Studio is open source software. See the LICENSE file for details.

---

**Simplicity through experience, not complexity through programming**
*AI Work Studio learns and adapts to become your perfect personal assistant*