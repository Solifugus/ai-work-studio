#!/bin/bash

# AI Work Studio Installation Script
# For Unix/Linux platforms
# Simplicity through experience, not complexity through programming

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="AI Work Studio"
REPO_URL="https://github.com/yourusername/ai-work-studio"
DEFAULT_INSTALL_DIR="$HOME/.local/bin"
DEFAULT_DATA_DIR="$HOME/.config/ai-work-studio"
DEFAULT_LOG_DIR="$HOME/.local/share/ai-work-studio/logs"

# Functions
print_header() {
    echo -e "${BLUE}================================"
    echo -e "  AI Work Studio Installer"
    echo -e "================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ Error: $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ Warning: $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

check_requirements() {
    print_info "Checking system requirements..."

    # Check Go installation
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        echo "Please install Go 1.21+ from https://golang.org/dl/"
        exit 1
    fi

    # Check Go version
    GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    REQUIRED_VERSION="1.21.0"
    if ! printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
        print_error "Go version $GO_VERSION found, but $REQUIRED_VERSION+ required"
        exit 1
    fi

    print_success "Go $GO_VERSION found"

    # Check for git
    if ! command -v git &> /dev/null; then
        print_error "Git is not installed or not in PATH"
        echo "Please install git using your package manager"
        exit 1
    fi

    print_success "Git found"

    # Check for make
    if ! command -v make &> /dev/null; then
        print_warning "Make not found. Will use go build directly"
    else
        print_success "Make found"
    fi
}

get_install_directory() {
    echo ""
    echo "Choose installation directory:"
    echo "1) $DEFAULT_INSTALL_DIR (default, user local)"
    echo "2) /usr/local/bin (system-wide, requires sudo)"
    echo "3) Custom directory"
    echo ""
    read -p "Selection [1]: " choice

    case ${choice:-1} in
        1)
            INSTALL_DIR="$DEFAULT_INSTALL_DIR"
            ;;
        2)
            INSTALL_DIR="/usr/local/bin"
            NEEDS_SUDO=true
            ;;
        3)
            read -p "Enter custom directory: " INSTALL_DIR
            if [[ ! "$INSTALL_DIR" = /* ]]; then
                INSTALL_DIR="$HOME/$INSTALL_DIR"
            fi
            ;;
        *)
            print_error "Invalid selection"
            exit 1
            ;;
    esac

    print_info "Will install to: $INSTALL_DIR"
}

get_data_directory() {
    echo ""
    read -p "Data directory [$DEFAULT_DATA_DIR]: " DATA_DIR
    DATA_DIR=${DATA_DIR:-$DEFAULT_DATA_DIR}

    print_info "Will use data directory: $DATA_DIR"
}

setup_directories() {
    print_info "Setting up directories..."

    # Create install directory
    if [[ "$NEEDS_SUDO" == "true" ]]; then
        sudo mkdir -p "$INSTALL_DIR"
    else
        mkdir -p "$INSTALL_DIR"
    fi

    # Create data directories
    mkdir -p "$DATA_DIR"/{nodes,edges,backups,cache,methods}
    mkdir -p "$DEFAULT_LOG_DIR"

    print_success "Directories created"
}

download_source() {
    print_info "Downloading source code..."

    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"

    if [[ -d ".git" ]]; then
        print_info "Using current directory (development mode)"
        BUILD_DIR="$(pwd)"
    else
        git clone "$REPO_URL" ai-work-studio
        cd ai-work-studio
        BUILD_DIR="$(pwd)"
    fi

    print_success "Source code ready"
}

build_project() {
    print_info "Building AI Work Studio..."

    # Download dependencies
    go mod download
    go mod tidy

    # Build binaries
    if command -v make &> /dev/null; then
        make build
        STUDIO_BIN="./bin/ai-work-studio"
        AGENT_BIN="./bin/ai-work-studio-agent"
    else
        # Fallback to direct go build
        mkdir -p bin
        go build -o bin/ai-work-studio ./cmd/studio
        go build -o bin/ai-work-studio-agent ./cmd/agent
        STUDIO_BIN="./bin/ai-work-studio"
        AGENT_BIN="./bin/ai-work-studio-agent"
    fi

    print_success "Build completed"
}

install_binaries() {
    print_info "Installing binaries..."

    if [[ "$NEEDS_SUDO" == "true" ]]; then
        sudo cp "$STUDIO_BIN" "$INSTALL_DIR/ai-work-studio"
        sudo cp "$AGENT_BIN" "$INSTALL_DIR/ai-work-studio-agent"
        sudo chmod +x "$INSTALL_DIR/ai-work-studio"
        sudo chmod +x "$INSTALL_DIR/ai-work-studio-agent"
    else
        cp "$STUDIO_BIN" "$INSTALL_DIR/ai-work-studio"
        cp "$AGENT_BIN" "$INSTALL_DIR/ai-work-studio-agent"
        chmod +x "$INSTALL_DIR/ai-work-studio"
        chmod +x "$INSTALL_DIR/ai-work-studio-agent"
    fi

    print_success "Binaries installed"
}

configure_system() {
    print_info "Setting up configuration..."

    # Create basic config file
    cat > "$DATA_DIR/config.json" << EOF
{
  "version": "1.0",
  "data_directory": "$DATA_DIR",
  "log_directory": "$DEFAULT_LOG_DIR",
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
EOF

    print_success "Basic configuration created"
}

setup_llm_config() {
    echo ""
    print_info "LLM Configuration Setup"
    echo "AI Work Studio supports both local and remote LLM providers."
    echo ""

    echo "Available options:"
    echo "1) Local models only (privacy-focused, no API costs)"
    echo "2) Anthropic Claude API (requires API key)"
    echo "3) OpenAI API (requires API key)"
    echo "4) Mixed (local + remote, configure later)"
    echo ""

    read -p "Select LLM setup [1]: " llm_choice

    case ${llm_choice:-1} in
        1)
            print_info "Local-only setup selected"
            ;;
        2)
            echo ""
            print_info "Anthropic Claude API setup"
            echo "Get your API key from: https://console.anthropic.com/"
            read -s -p "Enter Anthropic API key (will be hidden): " ANTHROPIC_KEY
            echo ""
            if [[ -n "$ANTHROPIC_KEY" ]]; then
                # Store in config (basic example - in production use secure storage)
                mkdir -p "$DATA_DIR/keys"
                echo "$ANTHROPIC_KEY" > "$DATA_DIR/keys/anthropic.key"
                chmod 600 "$DATA_DIR/keys/anthropic.key"
                print_success "Anthropic API key configured"
            fi
            ;;
        3)
            echo ""
            print_info "OpenAI API setup"
            echo "Get your API key from: https://platform.openai.com/api-keys"
            read -s -p "Enter OpenAI API key (will be hidden): " OPENAI_KEY
            echo ""
            if [[ -n "$OPENAI_KEY" ]]; then
                mkdir -p "$DATA_DIR/keys"
                echo "$OPENAI_KEY" > "$DATA_DIR/keys/openai.key"
                chmod 600 "$DATA_DIR/keys/openai.key"
                print_success "OpenAI API key configured"
            fi
            ;;
        4)
            print_info "Mixed setup selected - configure providers later in the UI"
            ;;
    esac
}

setup_systemd() {
    if [[ "$NEEDS_SUDO" == "true" ]] && command -v systemctl &> /dev/null; then
        echo ""
        read -p "Install systemd service for background agent? [y/N]: " install_service

        if [[ "$install_service" =~ ^[Yy]$ ]]; then
            print_info "Installing systemd service..."

            sudo tee /etc/systemd/system/ai-work-studio.service > /dev/null << EOF
[Unit]
Description=AI Work Studio Agent
After=network.target
Wants=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
ExecStart=$INSTALL_DIR/ai-work-studio-agent --config=$DATA_DIR/config.json
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
Environment=HOME=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF

            sudo systemctl daemon-reload
            print_success "Systemd service installed"
            print_info "Enable with: sudo systemctl enable ai-work-studio"
            print_info "Start with: sudo systemctl start ai-work-studio"
        fi
    fi
}

add_to_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo ""
        print_info "Adding $INSTALL_DIR to PATH..."

        # Add to shell profile
        for rc in ~/.bashrc ~/.zshrc ~/.profile; do
            if [[ -f "$rc" ]]; then
                echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$rc"
                print_info "Added to $rc"
                break
            fi
        done

        print_warning "Restart your shell or run: export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

run_tests() {
    echo ""
    read -p "Run quick verification tests? [Y/n]: " run_tests

    if [[ ! "$run_tests" =~ ^[Nn]$ ]]; then
        print_info "Running verification tests..."

        # Test binaries exist and are executable
        if [[ -x "$INSTALL_DIR/ai-work-studio" ]]; then
            print_success "Studio binary verified"
        else
            print_error "Studio binary not found or not executable"
        fi

        if [[ -x "$INSTALL_DIR/ai-work-studio-agent" ]]; then
            print_success "Agent binary verified"
        else
            print_error "Agent binary not found or not executable"
        fi

        # Test data directory structure
        if [[ -d "$DATA_DIR/nodes" && -d "$DATA_DIR/edges" ]]; then
            print_success "Data directory structure verified"
        else
            print_error "Data directory structure incomplete"
        fi

        # Test configuration
        if [[ -f "$DATA_DIR/config.json" ]]; then
            print_success "Configuration file verified"
        else
            print_error "Configuration file missing"
        fi
    fi
}

print_completion() {
    echo ""
    echo -e "${GREEN}================================"
    echo -e "  Installation Complete!"
    echo -e "================================${NC}"
    echo ""
    echo "Installation Summary:"
    echo "  Binaries:      $INSTALL_DIR/"
    echo "  Data:          $DATA_DIR"
    echo "  Logs:          $DEFAULT_LOG_DIR"
    echo "  Configuration: $DATA_DIR/config.json"
    echo ""
    echo "Next Steps:"
    echo "  1. Restart your shell or run: export PATH=\"$INSTALL_DIR:\$PATH\""
    echo "  2. Run: ai-work-studio --help"
    echo "  3. Start the GUI: ai-work-studio"
    echo "  4. Optional: Start background agent: ai-work-studio-agent"
    echo ""
    echo "Documentation: docs/installation.md"
    echo "Support: $REPO_URL/issues"
    echo ""
}

cleanup() {
    if [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]]; then
        cd /
        rm -rf "$TEMP_DIR"
    fi
}

# Main installation flow
main() {
    trap cleanup EXIT

    print_header

    # Check if already installed
    if command -v ai-work-studio &> /dev/null; then
        print_warning "AI Work Studio appears to already be installed"
        read -p "Continue with reinstallation? [y/N]: " continue_install
        if [[ ! "$continue_install" =~ ^[Yy]$ ]]; then
            echo "Installation cancelled"
            exit 0
        fi
    fi

    check_requirements
    get_install_directory
    get_data_directory
    setup_directories

    # Build or download
    if [[ -f "go.mod" && -f "Makefile" ]]; then
        print_info "Installing from current directory"
        BUILD_DIR="$(pwd)"
        build_project
    else
        download_source
        cd "$BUILD_DIR"
        build_project
    fi

    install_binaries
    configure_system
    setup_llm_config
    setup_systemd
    add_to_path
    run_tests
    print_completion
}

# Run with error handling
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi