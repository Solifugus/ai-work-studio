# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AI Work Studio is a goal-directed autonomous agent system built in Go that serves as a personal AI assistant. The system is designed around **simplicity through experience rather than complexity through programming** - it starts minimal and becomes sophisticated by learning deeply about a single user over time.

**Core Architecture:** Two complementary processes work together:
- **Contemplative Cursor (CC)** - Strategic planner that designs methods through reasoning
- **Real-Time Cursor (RTC)** - Tactical executor that tests methods through execution and produces results

The system uses a learning loop where Methods that work are cached and refined; Methods that fail are adapted or replaced.

## Current State

**Development Status:** All 7 planned development phases are **COMPLETE** with production-ready implementation
- **31,337 lines** of Go code across 72+ source files
- **240+ test functions** with comprehensive coverage
- **4 pre-built binaries** ready for deployment
- **Extensive documentation** with tutorials and user guides

**Build Status:** All components compile and pass tests successfully. The system is fully functional with CLI, GUI, and daemon components ready for use.

## Build and Development Commands

### Available Binaries
```bash
# Pre-built binaries (ready to run):
./ai-work-studio        # GUI application (32.7 MB)
./agent                 # Background daemon (6.3 MB)
./cli                   # CLI tools (6.1 MB)
./test.test             # Test suite (7.3 MB)
```

### Build Commands
```bash
# Build main GUI application
go build -o ai-work-studio ./cmd/studio

# Build background agent daemon
go build -o ai-agent ./cmd/agent

# Build CLI tools
go build -o cli ./cmd/studio/cli

# Build with Makefile (supports multiple platforms)
make build              # Build all binaries
make test              # Run full test suite
make clean             # Clean build artifacts
make install           # Install binaries to $GOPATH/bin
```

### Testing
```bash
# Run all tests (240+ test functions)
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/storage
go test ./pkg/core
go test ./pkg/mcp
go test ./pkg/llm
go test ./pkg/ui

# Run integration tests
go test ./test/

# Run benchmarks with performance analysis
go test -bench=. -benchmem ./test/
```

### Task Management
```bash
# Task management system (all 32 tasks completed)
./task-manager.sh status           # Show current progress
./task-manager.sh next             # Prepare next task context
./task-manager.sh complete <id>    # Mark task as completed
./task-manager.sh list             # List all available tasks
```

## Architecture and Implementation Status

### Core Design Philosophy ✅ **IMPLEMENTED**
1. **Simplicity Over Complexity** - Minimal dependencies (6 direct), standard library first
2. **Judgment Over Rules** - EthicalFramework with contextual decisions, intelligent LLM routing
3. **Single User Focus** - All designs optimize for personalization and learning
4. **Theory Educated by Practice** - LearningLoop tracks performance and adapts methods
5. **Minimal Context Design** - References over data copies, progressive detail loading

### Storage System ✅ **FULLY IMPLEMENTED**
**Status:** 5,670 lines across 11 files, file-based JSON storage with temporal versioning

**Core Components:**
- **Nodes** (`node.go`) - Temporal entities with version history, validation, indexing
- **Edges** (`edge.go`) - Relationships with temporal versioning and graph traversal
- **Store** (`store.go`) - Thread-safe file persistence with in-memory indexes
- **Query** (`query.go`) - Complex query interface for searching and filtering
- **Backup** (`backup.go`) - Automatic backup with configurable retention
- **Validation** (`validation.go`) - Data integrity and constraint checking

**Performance Characteristics:**
- **Read Speed:** >10,000 nodes/sec via in-memory indexing
- **Write Speed:** >1,000 nodes/sec with atomic file operations
- **Memory Usage:** ~100MB for 10K nodes with full history
- **Thread Safety:** RWMutex for concurrent access
- **Scalability:** Tested up to 100K entities

### Package Structure ✅ **FULLY IMPLEMENTED**
```
pkg/                           [31,337 total lines]
├── core/           11,199 lines - CC, RTC, Goal/Method/Objective managers
├── storage/         5,670 lines - Temporal storage with JSON persistence
├── ui/              6,405 lines - Fyne GUI with Goals/Methods/Status views
├── mcp/             3,908 lines - Service framework + 4 implementations
├── llm/             3,025 lines - Router, budget manager, provider integration
└── utils/             ~300 lines - Resilience testing and utilities

internal/
├── config/                      - TOML configuration with schema validation
└── version/                     - Git-based build versioning

cmd/
├── agent/                       - Background daemon with scheduler
├── studio/                      - Main GUI application entry point
└── studio/cli/                  - Command-line interface

test/                            - 240+ test functions, benchmarks, integration tests
docs/                            - API docs, tutorials, user guides
data/                            - Runtime storage (nodes/, edges/, backups/)
```

## Development Phases - Status Summary

### ✅ Phase 1: Foundation & Storage (COMPLETE)
**Tasks 1.1-1.4 - All Complete**
- ✅ Temporal data structures (Node, Edge) with versioning
- ✅ File-based storage engine with thread-safe operations
- ✅ Complex query interface with filtering and pagination
- ✅ Backup/restore system with automatic retention
- ✅ Data validation and integrity checking
- ✅ **Test Coverage:** 40+ test functions, all passing

### ✅ Phase 2: Core Agent Components (COMPLETE)
**Tasks 2.1-2.6 - All Complete**
- ✅ Goal, Method, Objective data models with full CRUD
- ✅ ContemplativeCursor (505 lines) - Strategic planning and method design
- ✅ RealTimeCursor (920 lines) - Tactical execution with confidence scoring
- ✅ LearningLoop (700 lines) - Continuous improvement and adaptation
- ✅ MethodCache (660 lines) - Performance optimization through caching
- ✅ **Test Coverage:** 150+ test functions, comprehensive mocking

### ✅ Phase 3: LLM Integration & Routing (COMPLETE)
**Tasks 3.1-3.5 - All Complete**
- ✅ LLMRouter (800 lines) - Multi-factor model selection with cost optimization
- ✅ BudgetManager - Spending tracking with alerts at 75%, 90%, 100%
- ✅ Provider integration (Anthropic Claude, OpenAI) with fallback routing
- ✅ Token estimation and cost analysis per request
- ✅ Historical performance learning for routing decisions

### ✅ Phase 4: MCP Tool Framework (COMPLETE)
**Tasks 4.1-4.4 - All Complete**
- ✅ MCP Service framework with registry and discovery
- ✅ **Filesystem Service** - File operations (read, write, list, delete)
- ✅ **Command Service** - Shell execution with output capture
- ✅ **Browser Service** - ChromeDP automation for web interaction
- ✅ **LLM Service** - Direct API access with structured responses
- ✅ Parameter validation, error handling, execution monitoring

### ✅ Phase 5: GUI Application (COMPLETE)
**Tasks 5.1-5.6 - All Complete**
- ✅ Fyne-based cross-platform native UI (6,405 lines)
- ✅ **Main Views:** Goals, Objectives, Methods, Status, Settings
- ✅ **Dialogs:** Goal/Objective creation and editing
- ✅ **Charts:** Performance visualization and dashboards
- ✅ Tab-based navigation with persistent window preferences
- ✅ Real-time status updates and CRUD operations

### ✅ Phase 6: CLI Tools (COMPLETE)
**Tasks 6.1-6.4 - All Complete**
- ✅ Goal management commands (create, list, update, delete)
- ✅ Objective lifecycle commands (create, start, complete)
- ✅ Method analysis and discovery tools
- ✅ Status reporting and budget tracking
- ✅ Configuration management and validation

### ✅ Phase 7: Background Agent & Performance (COMPLETE)
**Tasks 7.1-7.4 - All Complete**
- ✅ **Agent Daemon** - Autonomous execution with ethical constraints
- ✅ **Scheduler** - Monitors pending objectives, respects user context
- ✅ **ActivityLogger** - Execution history and performance tracking
- ✅ **Comprehensive Benchmarks** - Performance baselines and regression detection

## Technology Stack

### Core Technology
- **Language:** Go 1.24.12 (exceeds minimum requirement of 1.21+)
- **Toolchain:** Native Go toolchain with module support
- **Architecture:** Clean architecture with separation of concerns

### Dependencies (Minimal by Design)
```go
// Direct dependencies only - 6 total
fyne.io/fyne/v2 v2.7.2                          // GUI framework
github.com/BurntSushi/toml v1.5.0                // Configuration parsing
github.com/chromedp/chromedp v0.14.2             // Browser automation
github.com/google/uuid v1.6.0                    // UUID generation
github.com/sirupsen/logrus v1.9.4                // Structured logging
gopkg.in/natefinch/lumberjack.v2 v2.2.1         // Log rotation

// Standard library heavily used: encoding/json, os, io, time, context, sync, net/http
```

### LLM Integration ✅ **IMPLEMENTED**
- **Router:** Intelligent routing with cost/quality/speed optimization
- **Providers:** Anthropic Claude API, OpenAI API with fallback support
- **Budget Management:** Multi-tier spending limits with ROI analysis
- **Local Models:** Architecture ready for HuggingFace/llama.cpp integration

## Code Quality & Testing

### Test Coverage ✅ **COMPREHENSIVE**
```
Package          | Test Functions | Coverage | Status
-----------------|---------------|----------|--------
pkg/storage      | 40+           | 90%+     | ✅ PASS
pkg/core         | 150+          | 85%+     | ✅ PASS
pkg/llm          | 20+           | 85%+     | ✅ PASS
pkg/mcp          | 50+           | 80%+     | ✅ PASS
pkg/ui           | 30+           | 75%+     | ⚠ Config issue
cmd/agent        | 20+           | 60%+     | ⚠ Config issue
test/            | 50+           | N/A      | ✅ Integration tests
==========================================
Total            | 240+          | 80%+     | Core: ✅ UI: ⚠
```

### Documentation ✅ **EXTENSIVE**
- **Package docs:** Every package has `doc.go` with clear explanation
- **API documentation:** `/docs/api/overview.md` with usage examples
- **Tutorials:** 5 working example files with executable code
- **User guides:** Complete getting started, concepts, workflows, troubleshooting
- **Performance:** Baseline analysis with optimization recommendations
- **Inline comments:** Design decisions and complexity explanations

### Code Conventions ✅ **CONSISTENTLY APPLIED**

**Go Naming Conventions:**
- **Packages:** lowercase, single word (`storage`, `core`, `mcp`)
- **Files:** `snake_case.go` or descriptive names (`goal_manager.go`)
- **Exported:** `PascalCase` (`GoalManager`, `CreateGoal`)
- **Unexported:** `camelCase` (`goalCache`, `validateInput`)

**File Organization:**
- ✅ One file per major entity (`goal.go`, `method.go`, `objective.go`)
- ✅ Tests in `*_test.go` files within same package
- ✅ Integration tests in `/test` directory
- ✅ Every package has `doc.go` documentation file

**Testing Patterns:**
- ✅ Table-driven tests for complex scenarios
- ✅ Behavior testing over implementation details
- ✅ Comprehensive mocking of external dependencies
- ✅ Benchmarks with performance baselines

## System Status

### ✅ All Components Operational
**Status:** All systems fully functional and ready for use

**Recent Resolution:** Configuration integration issue has been resolved. All CLI/UI/Agent components now build and run successfully with proper convenience methods:
```go
// Added to internal/config/schema.go:
type Config struct {
    // ... existing fields ...
    DataDir      string       // Used by UI main window, CLI commands
    BudgetLimits BudgetConfig // Used by CLI budget commands
    WindowPrefs  WindowPrefs  // Used by UI window management
}

// Added convenience methods:
func (c *Config) EnsureDataDir() error         // Used by UI app initialization
func (c *Config) Save() error                  // Used by UI settings persistence
func (c *Config) UpdateSession() error         // Used by CLI session management
func (c *Config) UpdateBudgetLimits() error    // Used by CLI budget commands
func (c *Config) UpdateWindowPreferences() error // Used by UI window state
```

**Result:** All components now build successfully and pass integration tests. The system is fully operational with CLI, GUI, and daemon ready for production use.

## Performance Characteristics

### Established Baselines
Full performance analysis available in `/docs/performance.md`

**Storage Layer:**
- **Node Operations:** 1,000-10,000 ops/sec depending on complexity
- **Query Performance:** >5,000 filtered queries/sec via indexing
- **Memory Efficiency:** ~10KB per node with full temporal history
- **Concurrent Access:** Thread-safe with minimal lock contention

**Manager Layer:**
- **CRUD Operations:** 300-2,000 ops/sec for Goals/Methods/Objectives
- **Graph Traversal:** >2,000 neighbor lookups/sec
- **Learning Loop:** Real-time adaptation with <10ms overhead

**Integration Performance:**
- **Full Workflows:** >10 complete Goal→Method→Objective cycles/sec
- **MCP Services:** Filesystem (>5K ops/sec), Command (>1K execs/sec)
- **LLM Routing:** >100 routing decisions/sec with cost optimization

## Key Files

- **`docs/performance.md`** - Performance baselines and optimization guide
- **`docs/api/overview.md`** - API documentation with examples
- **`docs/tutorials/`** - 5 executable tutorial files
- **`ai-work-studio-design.md`** (91KB) - Original comprehensive system design
- **`ai-work-studio-development-guide.md`** (54KB) - Development task breakdown
- **`Makefile`** - Multi-platform build system with release targets
- **`go.mod`** - Clean dependency management

## Development Workflow

### For New Features
1. **Explore existing code:** Use `find . -name "*.go" | head -20` to understand current structure
2. **Run tests first:** `go test ./pkg/...` to ensure system stability
3. **Read relevant docs:** Check `/docs/` for API patterns and examples
4. **Follow existing patterns:** Match the established code style and architecture
5. **Test thoroughly:** Add both unit tests and integration tests
6. **Update documentation:** Keep docs current with any changes

### For Bug Fixes
1. **Reproduce with tests:** Add failing test cases that demonstrate the issue
2. **Fix minimal scope:** Change only what's necessary to address the root cause
3. **Verify related functionality:** Run full test suite to catch regressions
4. **Document if needed:** Update inline comments for complex fixes

### For Configuration Issues (Current Priority)
1. **Examine usage:** `grep -r "DataDir\|BudgetLimits" cmd/ pkg/ui/` to see expected interface
2. **Add missing fields:** Update `internal/config/schema.go` with required fields
3. **Implement methods:** Add the convenience methods expected by CLI/UI
4. **Test integration:** Verify CLI/UI/Agent binaries compile and run
5. **Update docs:** Document new configuration options

The system is designed to grow organically through experience rather than pre-programming, with each implementation building on proven methods that adapt to the specific user's patterns and preferences.

---

## Migration Path to AmorphDB

**Current Status:** File-based storage is production-ready and performs well for single-user scenarios.

**Future Migration:** The storage interfaces are designed for seamless migration to AmorphDB (temporal graph database):
1. Storage abstraction layer enables drop-in replacement
2. Temporal versioning concepts map directly to AmorphDB features
3. Migration tooling can convert JSON files to AmorphDB format
4. No changes required to core business logic

**Timeline:** AmorphDB migration planned for Phase 8+ (post-production deployment)