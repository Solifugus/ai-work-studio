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

## 🔄 System Workflow

Understanding how AI Work Studio organizes work:

### Core Concepts
1. **Goals** - High-level objectives you want to achieve
2. **Methods** - Proven approaches for accomplishing goals (learned by the system)
3. **Objectives** - Specific, actionable tasks within goals

### Workflow Process
```
📋 Goal Creation
    ↓
⚡ Objective Creation (within goal)
    ↓
🧠 Method Learning (automatic, based on success patterns)
    ↓
🔄 Continuous Improvement (methods evolve with experience)
```

### Typical Usage Pattern
```bash
# 1. Create a high-level goal
./ai-studio-cli create-goal "Learn Go Programming" "Master Go for backend development" 8

# 2. Create specific objectives for this goal
./ai-studio-cli create-objective <goal-id> "Complete Go tutorial" "Work through official Go tutorial"

# 3. System learns and suggests methods automatically as you work
# 4. Methods improve based on success/failure patterns
```

**Key Insight:** The system learns effective methods by observing which approaches lead to successful objective completion, building a personalized knowledge base over time.

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

1. **Set up your API keys:**
   ```bash
   export ANTHROPIC_API_KEY="your-key-here"
   export OPENAI_API_KEY="your-key-here"  # optional
   ```

2. **Create your first goal:**
   ```bash
   ./ai-studio-cli create-goal "Learn to use AI Work Studio effectively" "Master the core features and workflows" 7
   ```

3. **Launch interactive CLI mode:**
   ```bash
   ./ai-studio-cli
   # Type 'help' to see all available commands
   ```

4. **Launch the GUI:**
   ```bash
   ./ai-work-studio
   ```

### Interactive Mode

The CLI runs in interactive mode by default, providing a user-friendly command interface:

```bash
./ai-studio-cli
🤖 AI Work Studio - Interactive Mode
Type 'help' for commands, 'exit' to quit

ai-work-studio> help
# Shows complete command reference with examples

ai-work-studio> status
# Displays current goals and system status

ai-work-studio> exit
👋 Goodbye!
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

AI Work Studio uses smart defaults and requires minimal configuration to get started:

```bash
# The system automatically creates configuration on first run
# Default data directory: ~/.ai-work-studio/data
# Default config file: ~/.ai-work-studio/config.json
```

### Environment Variables (Recommended)

The easiest way to configure API access:

```bash
# Required for LLM functionality
export ANTHROPIC_API_KEY="your-anthropic-key-here"

# Optional for additional LLM providers
export OPENAI_API_KEY="your-openai-key-here"
```

### Command Line Configuration

```bash
# Override data directory
./ai-studio-cli -data /path/to/custom/data

# Use custom configuration file
./ai-studio-cli -config /path/to/config.json

# Enable verbose output
./ai-studio-cli -verbose
```

### Configuration File (Advanced)

Default configuration with examples:

```toml
[storage]
data_dir = "~/.ai-work-studio/data"
backup_enabled = true
backup_retention_days = 30

[api.anthropic]
api_key = ""  # Use ANTHROPIC_API_KEY env var instead
base_url = "https://api.anthropic.com"
default_model = "claude-3-sonnet-20241022"

[budget]
daily_limit = 5.00
monthly_limit = 150.00
per_request_limit = 0.50
tracking_enabled = true

[preferences]
auto_approve = false
verbose_output = false
default_priority = 5
interactive_mode = true
```

**Note:** The system creates configuration automatically with sensible defaults. Manual configuration is only needed for advanced customization.

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

**Command Line Interface:**
```bash
# Goal management
./ai-studio-cli create-goal "Complete project documentation" "Write comprehensive docs" 8
./ai-studio-cli list-goals
./ai-studio-cli status

# Objective tracking (requires goal ID from list-goals)
./ai-studio-cli create-objective <goal-id> "Write README.md" "Create comprehensive README documentation"
./ai-studio-cli list-objectives <goal-id>

# Configuration (limited keys supported)
./ai-studio-cli -data /custom/path    # Override data directory
./ai-studio-cli -verbose              # Enable verbose output
./ai-studio-cli -config /path/config  # Use custom config file
```

**Interactive Mode Commands:**
```bash
# Launch interactive mode
./ai-studio-cli

# Within interactive mode:
ai-work-studio> help                  # Show all commands
ai-work-studio> create-goal "Title" "Description" priority
ai-work-studio> list-goals            # List all goals
ai-work-studio> status                # Show system status
ai-work-studio> feedback "message"    # Provide system feedback
ai-work-studio> exit                  # Quit interactive mode
```

**Note:** The system follows a workflow where Goals contain Objectives. Methods are automatically managed by the learning system based on successful objective completions.

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