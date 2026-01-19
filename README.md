# AI Work Studio

> **A goal-directed autonomous agent system that learns and adapts to help you achieve your objectives**

[![Go Version](https://img.shields.io/badge/Go-1.22.2+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](#build-and-development)
[![Alpha Release](https://img.shields.io/badge/Release-v1.0.0--alpha-orange.svg)](https://github.com/Solifugus/ai-work-studio/releases)

## 🎯 Overview

AI Work Studio is a sophisticated personal AI assistant system built around **simplicity through experience rather than complexity through programming**. It starts minimal and becomes sophisticated by learning deeply about a single user over time through a unique dual-cursor architecture.

### Core Philosophy
- **Judgment Over Rules** - Makes contextual decisions rather than following rigid rules
- **Single User Focus** - Optimized for deep personalization and learning
- **Theory Educated by Practice** - Methods evolve based on execution results
- **Minimal Context Design** - Efficient data handling and reasonable token budgets

## 🏗️ Architecture

**Dual-Cursor Design:**
- **🧠 Contemplative Cursor (CC)** - Strategic planner that designs methods through reasoning
- **⚡ Real-Time Cursor (RTC)** - Tactical executor that tests methods and produces results

The system uses a learning loop where Methods that work are cached and refined; Methods that fail are adapted or replaced.

## ✨ Key Features

### 🎯 Goal Management
- **Hierarchical goal structure** with dependencies and priorities
- **Temporal versioning** for tracking goal evolution over time
- **Smart scheduling** based on user patterns and preferences

### 🔄 Learning System
- **Method cache** with semantic similarity matching
- **Success/failure tracking** for continuous improvement
- **Adaptive scheduling** based on historical performance

### 🛠️ Tool Integration
- **MCP (Model Context Protocol)** framework for extensible tool support
- **Built-in tools**: filesystem, browser automation, command execution
- **Custom tool development** with comprehensive SDK

### 🤖 LLM Integration
- **Multi-provider support**: Anthropic Claude, OpenAI, local models
- **Intelligent routing** based on task complexity and cost
- **Budget management** with daily/monthly spending limits

### 📊 Performance Monitoring
- **Real-time performance metrics** and benchmarking
- **Resource usage tracking** with optimization recommendations
- **Comprehensive logging** and debugging support

## 🚀 Quick Start

### Installation

#### Option 1: Quick Install Script
```bash
curl -sSL https://raw.githubusercontent.com/Solifugus/ai-work-studio/master/install.sh | bash
```

#### Option 2: Manual Installation
```bash
# Clone the repository
git clone https://github.com/Solifugus/ai-work-studio.git
cd ai-work-studio

# Build all components
make build

# Install (optional)
make install
```

### First Run

1. **Initialize configuration:**
   ```bash
   ./ai-studio-cli config init
   ```

2. **Set up your API keys:**
   ```bash
   export ANTHROPIC_API_KEY="your-key-here"
   export OPENAI_API_KEY="your-key-here"  # optional
   ```

3. **Create your first goal:**
   ```bash
   ./ai-studio-cli goal create "Learn to use AI Work Studio effectively"
   ```

4. **Launch the GUI:**
   ```bash
   ./ai-work-studio
   ```

## 🏗️ Build and Development

### Prerequisites
- **Go 1.22.2+**
- **Git**
- **Make** (optional, for build automation)

### Available Components

```bash
# Build all components
make build

# Individual builds
go build -o ai-work-studio ./cmd/studio      # GUI application
go build -o ai-studio-cli ./cmd/studio/cli   # Command-line interface
go build -o ai-agent ./cmd/agent             # Background daemon
```

### Development Commands

```bash
# Run tests
make test
go test ./...

# Run benchmarks with performance analysis
make bench
go test -bench=. -benchmem ./test/

# Generate documentation
make docs
./docs/generate_docs.sh

# Clean build artifacts
make clean
```

## 📁 Project Structure

```
ai-work-studio/
├── cmd/                    # Executable applications
│   ├── agent/             # Background agent daemon
│   ├── studio/            # GUI application (Fyne)
│   └── studio/cli/        # Command-line interface
├── pkg/                    # Core packages
│   ├── core/              # Goal, Method, Objective, Cursors
│   ├── storage/           # Temporal graph storage engine
│   ├── llm/               # LLM routing and budget management
│   ├── mcp/               # Model Context Protocol tools
│   └── ui/                # Fyne GUI components
├── internal/              # Private packages
│   └── config/            # Configuration management
├── test/                  # Integration tests and benchmarks
├── docs/                  # Documentation and tutorials
├── config/                # Configuration examples
└── systemd/               # System service files
```

## ⚙️ Configuration

### Basic Setup
```bash
# Create configuration directory
mkdir -p ~/.ai-work-studio

# Copy example configuration
cp config/config.example.toml ~/.ai-work-studio/config.toml

# Edit configuration
nano ~/.ai-work-studio/config.toml
```

### Key Configuration Options

```toml
[storage]
data_dir = "~/.ai-work-studio/data"
backup_enabled = true

[api.anthropic]
api_key = ""  # Or set ANTHROPIC_API_KEY env var
default_model = "claude-3-sonnet-20241022"

[budget]
daily_limit = 5.00
monthly_limit = 150.00
tracking_enabled = true

[preferences]
auto_approve = false
verbose_output = false
interactive_mode = true
```

## 📊 Performance Characteristics

**Established Baselines:**
- **Storage Layer**: >10K reads/sec, <10ms P95 latency
- **Manager Layer**: >500 creates/sec, <20ms P95 latency
- **Method Cache**: >20K gets/sec, <1ms P95 latency
- **LLM Layer**: >100 assessments/sec, <100ms P95 latency

**System Requirements:**
- **Minimum**: 2+ cores, 4GB RAM, 1GB storage
- **Recommended**: 4+ cores, 8GB+ RAM, 10GB+ storage (SSD)

## 🔧 Usage Examples

### CLI Operations
```bash
# Goal management
./ai-studio-cli goal create "Complete project documentation"
./ai-studio-cli goal list --status active
./ai-studio-cli goal show <goal-id>

# Method operations
./ai-studio-cli method create --goal-id <goal-id> "Break into smaller tasks"
./ai-studio-cli method list --successful-only

# Objective tracking
./ai-studio-cli objective create --method-id <method-id> "Write README.md"
./ai-studio-cli objective start <objective-id>
./ai-studio-cli objective complete <objective-id>

# Configuration
./ai-studio-cli config set budget.daily_limit 10.00
./ai-studio-cli config get preferences.verbose_output
```

### Programmatic Usage
```go
package main

import (
    "github.com/Solifugus/ai-work-studio/pkg/core"
    "github.com/Solifugus/ai-work-studio/pkg/storage"
)

func main() {
    // Initialize storage
    store, _ := storage.NewStore("./data")

    // Create goal manager
    goalMgr := core.NewGoalManager(store)

    // Create a new goal
    goal, _ := goalMgr.CreateGoal(core.Goal{
        Title:       "Learn Go programming",
        Description: "Master Go for system development",
        Priority:    5,
    })

    // Goal is now tracked in the system
    fmt.Printf("Created goal: %s\n", goal.ID)
}
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
```bash
# Clone and setup
git clone https://github.com/Solifugus/ai-work-studio.git
cd ai-work-studio

# Install dependencies
go mod download

# Run tests to verify setup
go test ./...

# Make changes and test
go test -race ./...
go test -bench=. ./test/
```

### Code Style
- Follow standard Go conventions
- Use `gofmt` and `golint`
- Write tests for new features
- Update documentation for user-facing changes

## 📋 Roadmap

### Current (v1.0.0-alpha)
- ✅ Core goal/method/objective system
- ✅ Temporal storage with file-based persistence
- ✅ LLM integration with multiple providers
- ✅ MCP tool framework
- ✅ Cross-platform GUI and CLI

### Planned (v1.1.0)
- 🔄 AmorphDB migration for enhanced performance
- 🔄 Advanced learning algorithms
- 🔄 Plugin system for custom tools
- 🔄 Web interface option
- 🔄 Cloud deployment support

### Future (v2.0.0)
- 🔮 Multi-user support
- 🔮 Distributed agent networks
- 🔮 Advanced AI reasoning capabilities
- 🔮 Integration marketplace

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Claude by Anthropic** for AI assistance and guidance
- **Fyne** for the cross-platform GUI framework
- **Go community** for excellent tooling and libraries
- **MCP Protocol** for extensible tool integration

## 📞 Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/Solifugus/ai-work-studio/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Solifugus/ai-work-studio/discussions)

---

**🌟 Star this project if you find it useful!**

*AI Work Studio - Where goals meet intelligence.*