# AI Work Studio: Development Task Guide

**Version:** 1.0  
**Updated:** January 2026  
**Purpose:** Step-by-step development tasks for Claude Coder sessions with minimal token overhead

---

## Development Philosophy & Conventions

### Important Note on Storage

**Current Approach:** This initial implementation uses a file-based storage system with temporal versioning. Each node and edge is stored as a JSON file, with version history maintained in-file. This is simple, requires no external dependencies, and sufficient for a single-user system.

**Future Migration:** The storage layer is designed to support migration to AmorphDB (a temporal graph database currently in development) when ready. The abstractions (Node, Edge, Query interface) match AmorphDB's design, making migration straightforward. When AmorphDB is ready:
1. Implement AmorphDB backend for storage interface
2. Migrate data using conversion tool
3. Swap implementation - no changes needed to core logic

For now, focus on getting the file-based system working well. The migration path is clear and won't require redesigning the core components.

---

### Core Principles (Read Before Starting ANY Task)

**1. Simplicity Over Complexity**
- Use minimal dependencies - if you can solve it with stdlib, don't add a library
- Prefer general solutions over specific implementations
- If a design choice adds special cases or complexity, it's probably wrong
- Each component should be explainable in 2-3 sentences
- Test: Can you explain this module to someone in 5 minutes? If no, simplify.

**2. Judgment Over Rules**
- Don't hardcode specific behaviors or thresholds
- Build systems that make contextual decisions
- Avoid rigid if/then logic chains
- Use reasoning and evaluation, not rule engines

**3. Single User Focus**
- Every design decision optimizes for one user, not many
- Personalization comes from learning, not configuration
- No multi-tenant concerns needed
- Each instance is unique and evolves differently

**4. Theory Educated by Practice**
- Systems learn from execution results
- Failed methods should be replaced or refined toward simplicity
- Beauty test: modifications should reduce complexity, not increase it
- Track what was tried, what failed, what works

**5. Minimal Context Design**
- Pass references to data, not the data itself
- Include only essential context for each task
- Full context available on demand, not by default
- Keep token budgets reasonable

### Technology Stack

**Required:**
- **Language:** Go 1.21+ (standard library first)
- **UI Framework:** Fyne (native cross-platform GUI)
- **Storage:** Local JSON files with temporal versioning (AmorphDB integration planned for future)
- **Protocol:** MCP (Model Context Protocol) for tool integration
- **LLM Access:** 
  - Local models: HuggingFace models (via transformers/llama.cpp)
  - Remote APIs: Anthropic Claude API, OpenAI API

**Standard Library Usage:**
- `encoding/json` for serialization
- `os` and `io` for file operations
- `time` for temporal operations
- `context` for cancellation and timeouts
- `sync` for concurrency primitives
- `net/http` for API calls

**Essential External Libraries (minimal set):**
- `fyne.io/fyne/v2` - GUI framework
- `github.com/google/uuid` - UUID generation
- HuggingFace integration (TBD - evaluate options like `go-gpt3` or direct HTTP)
- Anthropic/OpenAI SDK or direct HTTP client

**Avoid:**
- Heavy frameworks or ORMs
- Complex dependency trees
- Vendor lock-in where possible
- Cgo unless absolutely necessary (for performance)

### Code Conventions

**File Organization:**
```
ai-work-studio/
├── cmd/
│   ├── agent/          # Background agent daemon
│   └── studio/         # Main GUI application
├── pkg/
│   ├── core/           # CC, RTC, objective management
│   ├── storage/        # File-based storage with temporal versioning
│   ├── mcp/            # MCP tool implementations
│   ├── llm/            # LLM routing and providers
│   └── ui/             # Fyne UI components
├── internal/           # Internal packages
│   └── config/         # Configuration management
├── test/               # Integration tests
├── data/               # Default data directory (gitignored)
├── docs/               # Documentation
└── configs/            # Configuration examples (not secrets)
```

**Go Naming Conventions:**
- Packages: lowercase, single word when possible
- Files: `snake_case.go` or `lowercase.go`
- Exported: `PascalCase` (capitalized first letter)
- Unexported: `camelCase` (lowercase first letter)
- Interfaces: Often just name the behavior (e.g., `Reader`, `Writer`)
- Constants: `PascalCase` or `UPPER_CASE` for package-level

**Documentation:**
- Every package: `doc.go` file with package overview
- Every exported type: godoc comment explaining purpose
- Every exported function: godoc comment with params and behavior
- Complex logic: inline comments explaining *why*, not *what*
- Follow godoc conventions (complete sentences, starts with symbol name)

**Testing:**
- Each package gets `*_test.go` files
- Use Go's built-in testing package
- Table-driven tests for multiple cases
- Test the behavior, not the implementation
- Mock external dependencies (LLM calls, file system when appropriate)
- Integration tests in `/test` directory

**Error Handling:**
- Fail gracefully with informative messages
- Log errors with context for debugging
- Don't catch exceptions you can't handle meaningfully
- Provide recovery paths when possible

### Design Patterns to Follow

**1. Minimal Interfaces:**
```go
// Good: Minimal, focused interface
type Method interface {
    Execute(ctx context.Context, input MinimalContext) (Result, error)
}

// Bad: Kitchen sink interface
type Method interface {
    Execute(fullProfile, allGoals, completeHistory, ...) error
}
```

**2. Composition Over Inheritance:**
```go
// Good: Compose behaviors (Go has no inheritance)
type Objective struct {
    goalID      string
    methodCache MethodCache
}

// Go uses embedding for composition
type EnhancedObjective struct {
    Objective
    extraField string
}
```

**3. Data References Over Data Copies:**
```go
// Good: Reference to data location
task := Task{
    GoalRef:   "goal_123",
    InputRefs: []string{"file://path/to/data.json"},
}

// Bad: Embed full data in task
task := Task{
    GoalData:  entireGoalStruct,
    InputData: hugeDataStructure,
}
```

**4. Progressive Detail:**
```go
// High-level plan (minimal context)
plan := []Task{
    {ID: "1", Type: "gather_data", Context: map[string]any{"what": "financials"}},
    {ID: "2", Type: "analyze", Context: map[string]any{"ref": "task_1_output"}},
}

// Detail loaded on demand when task executes
func executeTask(taskID string) error {
    task := getTask(taskID)
    fullContext := loadContextForTask(task)  // Only when needed
    // ...
}
```

**5. Error Handling (Go idioms):**
```go
// Return errors, don't panic (except for truly exceptional cases)
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)  // Wrap errors for context
}

// Use sentinel errors for specific cases
var ErrNotFound = errors.New("not found")
```

### Common Pitfalls to Avoid

1. **Over-architecting:** Don't build for hypothetical future requirements
2. **Premature optimization:** Build it working, then make it fast if needed
3. **Feature creep:** Stick to the minimal viable implementation
4. **External dependencies:** Question every `pip install` - is it truly needed?
5. **Configuration complexity:** Prefer convention over configuration
6. **Rigid abstractions:** Keep things flexible and judgment-based
7. **Token waste:** Always consider token budget in designs

---

## Phase 1: Foundation & Storage

### Task 1.1: Temporal Data Structures

**Context:** The system needs to store entities (goals, methods, objectives) with full temporal history. Every modification creates a new version; nothing is deleted. For now, we'll use local JSON files instead of AmorphDB, but design the structures to support temporal queries and future migration.

**Objective:** Implement the core temporal data structures that will be persisted as JSON.

**Requirements:**
- `Node` struct: ID, Type, Data (map), CreatedAt, ValidFrom, ValidUntil
- `Edge` struct: ID, SourceID, TargetID, Type, Data (map), CreatedAt, ValidFrom, ValidUntil
- Both support temporal queries: "get version at timestamp"
- Version immutability: once created, never modified in JSON
- ValidUntil = nil/zero means "current/active version"

**Key Design Decisions:**
- Use Go structs with JSON tags
- Use `time.Time` (RFC3339 format in JSON)
- ValidUntil == time.Time zero value means current
- All data payloads are `map[string]interface{}` (JSON-serializable)
- Use `github.com/google/uuid` for IDs

**Files to Create:**
- `pkg/storage/node.go`
- `pkg/storage/edge.go`
- `pkg/storage/node_test.go`
- `pkg/storage/edge_test.go`

**Success Criteria:**
- Can create nodes and edges with temporal metadata
- Can marshal/unmarshal to/from JSON
- Can query node/edge state at any timestamp
- Version history is preserved
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

### Task 1.2: File-Based Storage Engine

**Context:** The storage engine manages collections of nodes and edges, provides querying capabilities, and handles version management. Everything is persisted as JSON files in a structured directory.

**Objective:** Build the storage engine that manages nodes and edges with file-based persistence.

**Requirements:**
- Store nodes and edges as separate JSON files
- Directory structure: `data/nodes/{type}/{id}.json`, `data/edges/{id}.json`
- In-memory indexes for performance (load on startup)
- Support operations: AddNode, UpdateNode, GetNode, AddEdge, UpdateEdge, GetEdge
- Temporal queries: GetNodeAtTime, GetEdgesAtTime
- Graph traversal: GetNeighbors, GetEdgesByType
- Auto-save on modifications (with write-through)

**Key Design Decisions:**
- One JSON file per entity (all versions in one file as array)
- In-memory maps: `map[string][]Node` for quick access
- File locking for concurrent access (use `sync.RWMutex`)
- Atomic writes: temp file + rename
- Load all on startup (acceptable for single-user)

**Files to Create:**
- `pkg/storage/store.go`
- `pkg/storage/store_test.go`

**Dependencies:**
- Requires: Task 1.1 completed (Node, Edge structs)

**Success Criteria:**
- Can store and retrieve nodes/edges
- Changes persist to disk immediately
- Temporal queries work correctly
- Graph traversal works
- Concurrent access is safe
- All tests pass

**Estimated Lines:** ~350-400 lines total

---

### Task 1.3: Query Interface

**Context:** Provide a high-level query interface for common operations like finding nodes by type, traversing relationships, and temporal range queries. This abstracts the storage engine details.

**Objective:** Implement a fluent query interface for the storage system.

**Requirements:**
- Query builder: filter by type, properties, time range
- Relationship traversal: follow edges by type
- Temporal operations: AsOf(timestamp), Between(start, end)
- Support chaining: `store.Nodes().OfType("Goal").AsOf(yesterday).All()`
- Return slices or iterators appropriately

**Key Design Decisions:**
- Builder pattern for query construction
- Lazy evaluation where practical
- Filter in Go (simple, no query language needed)
- Return `[]Node` or `[]Edge` for results
- Keep it simple: linear scans are fine for single-user data

**Files to Create:**
- `pkg/storage/query.go`
- `pkg/storage/query_test.go`

**Dependencies:**
- Requires: Task 1.2 completed (Storage engine)

**Success Criteria:**
- Can query nodes/edges by various criteria
- Temporal queries work correctly
- Relationship traversal works
- Fluent API is intuitive
- All tests pass

**Estimated Lines:** ~250-300 lines total

---

### Task 1.4: Storage Reliability & Recovery

**Context:** Need to ensure data integrity and recover from corruption or crashes. Implement validation, backups, and recovery mechanisms.

**Objective:** Add reliability features to the storage system.

**Requirements:**
- Validation on load: check JSON structure, required fields
- Automatic backups: periodic snapshots to `data/backups/`
- Corruption detection: checksums or validation
- Recovery: restore from last good backup
- Graceful degradation: skip corrupted files, log errors

**Key Design Decisions:**
- Validate JSON schema on load using struct tags
- Backups: copy entire data directory periodically
- Keep last N backups (configurable, default 10)
- On corruption: log error, attempt partial recovery
- Health check function: validate all data files

**Files to Create:**
- `pkg/storage/validation.go`
- `pkg/storage/backup.go`
- `pkg/storage/validation_test.go`
- `pkg/storage/backup_test.go`

**Dependencies:**
- Requires: Task 1.2 completed (Storage engine)

**Success Criteria:**
- Validates data on load
- Creates backups automatically
- Can recover from corrupted files
- Health check works
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

## Phase 2: Core Agent Components

### Task 2.1: Goal Data Model

**Context:** Goals are the user's objectives that the system serves. They have hierarchies (goals serve other goals), evolve over time, and can be active, paused, or completed. Goals are stored as nodes in the file-based storage.

**Objective:** Define the Goal data model and basic operations.

**Requirements:**
- Goal struct with: ID, Title, Description, Status, Priority, UserContext, CreatedAt
- Status: active, paused, completed, archived (use typed constants/enum)
- Priority: numeric (1-10) or contextual judgment
- Hierarchy: goals can serve other goals (edge relationship)
- Temporal evolution: description/priority can change over time

**Key Design Decisions:**
- Store as storage.Node with Type="Goal"
- Data payload contains: title, description, status, priority, userContext (as map)
- Hierarchy via edges: Type="serves", Source=subgoal, Target=parentGoal
- Status changes create new versions
- Don't hardcode priority meanings - system learns what they mean
- Helper methods for common operations (Create, Update, Get, etc.)

**Files to Create:**
- `pkg/core/goal.go`
- `pkg/core/goal_test.go`

**Dependencies:**
- Requires: Task 1.3 completed (Storage query interface)

**Success Criteria:**
- Can create, update, query goals
- Hierarchy relationships work
- Status transitions tracked temporally
- All tests pass

**Estimated Lines:** ~250-300 lines total

---

### Task 2.2: Method Data Model

**Context:** Methods are approaches for achieving objectives. They're learned through experience, evolved over time, and cached when proven. Methods can be general (universal), domain-specific, or user-specific. They're stored with version history.

**Objective:** Define the Method data model with versioning support.

**Requirements:**
- Method attributes: id, name, description, approach, domain, version, created_at
- Approach: structured representation (e.g., steps, tools, heuristics)
- Domain: general, domain-specific, user-specific
- Versioning: track evolution from prior versions
- Success tracking: execution count, success rate, last used

**Key Design Decisions:**
- Store as AmorphDB node with type="Method"
- Evolution via edges: type="evolved_from", source=new_version, target=old_version
- Approach stored as structured data (dict/list of steps)
- Success metrics updated via Execution results
- Methods never deleted, only marked deprecated or superseded

**Files to Create:**
- `pkg/core/method.go`
- `test/method.go`

**Dependencies:**
- Requires: Task 1.3 completed (AmorphDB query interface)

**Success Criteria:**
- Can create, version, query methods
- Evolution tracking works
- Success metrics updated correctly
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

### Task 2.3: Objective Data Model

**Context:** Objectives are specific tasks to achieve goals. They have a method (approach), minimal context for execution, and track their lifecycle from creation to completion. Results inform method refinement.

**Objective:** Define the Objective data model and lifecycle management.

**Requirements:**
- Objective attributes: id, goal_id, method_id, context, status, created_at, completed_at
- Status: pending, in_progress, completed, failed, paused
- Context: minimal necessary data for execution (references preferred)
- Result: outcome data when completed
- Lifecycle events: creation, start, completion, failure

**Key Design Decisions:**
- Store as AmorphDB node with type="Objective"
- Links to Goal via edge: type="serves"
- Links to Method via edge: type="uses"
- Minimal context stored: references to data, not data itself
- Results stored as new version when objective completes
- Track token usage for budget management

**Files to Create:**
- `pkg/core/objective.go`
- `test/objective.go`

**Dependencies:**
- Requires: Task 2.1, 2.2 completed (Goal, Method models)

**Success Criteria:**
- Can create, track, complete objectives
- Lifecycle state management works
- Relationships to goals/methods tracked
- All tests pass

**Estimated Lines:** ~250-300 lines total

---

### Task 2.4: Contemplative Cursor (CC) - Basic Planning

**Context:** The CC is the strategic planner. It receives high-level objectives, designs methods, and creates execution plans. It operates with broad context but produces minimal-context plans for execution. This task implements basic planning capability.

**Objective:** Implement CC's core planning logic that creates execution plans from objectives.

**Requirements:**
- Input: Objective with goal context
- Process: Reason about approach, select or design method
- Output: Execution plan (sequence of tasks with minimal contexts)
- Method selection: query cache for proven methods first
- Method design: create new method if none cached
- Plan decomposition: break complex objectives into subtasks

**Key Design Decisions:**
- CC uses LLM for reasoning (via MCP service)
- Check method cache first (AmorphDB query)
- If new method needed, design based on general strategies
- Decompose recursively if needed (token budget aware)
- Plan includes: task sequence, dependencies, contexts
- Don't execute - just plan

**Files to Create:**
- `pkg/core/contemplative_cursor.go`
- `test/contemplative_cursor.go`

**Dependencies:**
- Requires: Task 2.1, 2.2, 2.3 completed
- Requires: MCP service for LLM access (can mock for testing)

**Success Criteria:**
- Can create execution plans for objectives
- Queries method cache appropriately
- Designs new methods when needed
- Plans use minimal context
- All tests pass (with mocked LLM)

**Estimated Lines:** ~300-400 lines total

---

### Task 2.5: Real-Time Cursor (RTC) - Basic Execution

**Context:** The RTC is the tactical executor. It takes execution plans, runs tasks, uses tools, and produces results. It operates with minimal context per task, requesting more detail only when needed. This task implements basic execution capability.

**Objective:** Implement RTC's core execution logic that runs tasks from plans.

**Requirements:**
- Input: Execution plan from CC
- Process: Execute tasks sequentially, use tools via MCP
- Output: Results for each task, overall objective outcome
- Tool usage: call MCP services as needed
- Error handling: retry logic, escalation to CC if stuck
- Context loading: fetch full context only when task needs it

**Key Design Decisions:**
- RTC uses LLM for task execution (via MCP service)
- Each task gets minimal context from plan
- Tools accessed via MCP protocol
- Failed tasks can request more context or new plan from CC
- Results captured for method refinement
- Track token usage per task

**Files to Create:**
- `pkg/core/real_time_cursor.go`
- `test/real_time_cursor.go`

**Dependencies:**
- Requires: Task 2.4 completed (CC planning)
- Requires: MCP services (can mock for testing)

**Success Criteria:**
- Can execute plans from CC
- Uses tools via MCP
- Handles errors gracefully
- Tracks results properly
- All tests pass (with mocked LLM/tools)

**Estimated Lines:** ~300-400 lines total

---

### Task 2.6: CC-RTC Integration & Learning Loop

**Context:** CC and RTC work together in a learning loop: CC plans, RTC executes, results inform CC's method refinement. Successful executions strengthen methods; failures trigger method evolution.

**Objective:** Implement the integration between CC and RTC with learning feedback.

**Requirements:**
- Execution flow: Objective → CC plan → RTC execute → Results → CC refine
- Success tracking: update method success metrics
- Failure handling: CC analyzes failure, modifies or replaces method
- Method evolution: create new version when refined
- Caching: successful methods cached for reuse
- Simplification bias: refinements should reduce complexity

**Key Design Decisions:**
- After execution, RTC returns results to CC
- CC evaluates: success, partial success, or failure
- If success: increment method success count, cache if proven
- If failure: CC analyzes why, decides modify vs replace
- Beauty test: refinements that add complexity probably wrong
- New method version links to old via "evolved_from" edge

**Files to Create:**
- `pkg/core/learning_loop.go`
- `test/learning_loop.go`

**Dependencies:**
- Requires: Task 2.4, 2.5 completed (CC, RTC)
- Requires: Task 2.2 completed (Method versioning)

**Success Criteria:**
- CC-RTC loop executes correctly
- Methods evolve based on results
- Success tracking works
- Learning improves methods over time
- All tests pass

**Estimated Lines:** ~250-300 lines total

---

## Phase 3: MCP Services & Tools

### Task 3.1: MCP Service Framework

**Context:** MCP (Model Context Protocol) is how the system interacts with external tools and capabilities. Need a framework for building MCP services that the RTC can call during execution.

**Objective:** Build the MCP service framework and registry.

**Requirements:**
- Service interface: standard methods all services implement
- Registry: discover and call services by name
- Parameter validation: ensure correct params passed
- Error handling: graceful failures with informative errors
- Logging: track service calls for debugging

**Key Design Decisions:**
- Simple abstract base class for services
- Registry is dict of name → service instance
- Services return structured results (success/failure, data, error)
- Use Python's built-in `abc` module
- Keep it simple: sync for now, async later if needed

**Files to Create:**
- `pkg/mcp/framework.go`
- `pkg/mcp/registry.go`
- `test/mcp_framework.go`

**Success Criteria:**
- Can define services with standard interface
- Registry manages service instances
- Services can be called with validation
- Errors handled properly
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

### Task 3.2: File System MCP Service

**Context:** One of the most basic capabilities is file system access. The RTC needs to read/write files, list directories, check existence, etc. This is done through an MCP service.

**Objective:** Implement file system operations as an MCP service.

**Requirements:**
- Operations: read_file, write_file, list_directory, exists, create_directory, delete_file
- Path validation: ensure paths are within allowed directories
- Error handling: file not found, permission denied, etc.
- Safety: confirm before destructive operations (based on context)

**Key Design Decisions:**
- Use filepath package for all path operations
- Restrict access to configured base directories (security)
- Read/write in chunks for large files
- Return structured results (success, data/error, metadata)
- Text vs binary mode based on file extension or parameter

**Files to Create:**
- `pkg/mcp/filesystem.go`
- `test/mcp_filesystem.go`

**Dependencies:**
- Requires: Task 3.1 completed (MCP framework)

**Success Criteria:**
- All file operations work correctly
- Path restrictions enforced
- Errors handled gracefully
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

### Task 3.3: LLM Access MCP Service

**Context:** The RTC needs to call LLMs for task execution. This should be abstracted through an MCP service that handles provider selection, budget tracking, and result formatting.

**Objective:** Implement LLM access as an MCP service.

**Requirements:**
- Support multiple providers: Anthropic Claude API, OpenAI API, local HuggingFace models
- Operations: Complete (text completion), Embed (get embeddings)
- Budget tracking: token count, cost calculation
- Provider selection: based on task requirements
- Error handling: rate limits, API failures, retries

**Key Design Decisions:**
- Abstract provider differences behind common interface
- For remote APIs: use official SDKs or direct HTTP with `net/http`
- For local HuggingFace models: evaluate options:
  - Option 1: HTTP to local inference server (llama.cpp, text-generation-webui)
  - Option 2: Direct bindings (if Go library exists)
  - Option 3: Subprocess to Python script (fallback)
- Track tokens and costs for budget management
- Simple provider selection for now (explicit choice)
- Return: text, token count, cost, provider used
- Use environment variables for API keys (not in code)
- Config specifies which models available locally vs remotely

**Files to Create:**
- `pkg/mcp/llm.go`
- `test/mcp_llm.go` (with mocked API calls)

**Dependencies:**
- Requires: Task 3.1 completed (MCP framework)

**Success Criteria:**
- Can call different LLM providers
- Budget tracking works
- Errors handled with retries
- All tests pass (mocked)

**Estimated Lines:** ~300-350 lines total

---

### Task 3.4: Web Browser MCP Service (Basic)

**Context:** For tasks like research, monitoring, or web interaction, need browser automation. Start with basic operations: navigate, extract text, click elements.

**Objective:** Implement basic web browser automation as an MCP service.

**Requirements:**
- Operations: navigate(url), get_text(), get_element(selector), click(selector)
- Headless operation (no GUI)
- Timeout handling
- JavaScript execution support
- Screenshot capability

**Key Design Decisions:**
- Use chromedp or rod for browser automation
- Headless by default
- Return structured content (not raw HTML)
- Handle dynamic content (wait for load)
- Rate limiting to be respectful

**Files to Create:**
- `pkg/mcp/browser.go`
- `test/mcp_browser.go` (integration tests)

**Dependencies:**
- Requires: Task 3.1 completed (MCP framework)
- External: `github.com/chromedp/chromedp` or `github.com/go-rod/rod`

**Success Criteria:**
- Can navigate and extract content
- Handles dynamic pages
- Errors handled gracefully
- All tests pass

**Estimated Lines:** ~250-300 lines total

---

### Task 3.5: Command Execution MCP Service

**Context:** For development tasks, system operations, and tool usage, need ability to execute shell commands safely.

**Objective:** Implement command execution as an MCP service with safety controls.

**Requirements:**
- Execute shell commands
- Capture stdout, stderr, exit code
- Timeout support
- Working directory control
- Command allowlist (safety)

**Key Design Decisions:**
- Use os/exec package
- Require explicit approval for destructive commands
- Timeout default: 30 seconds
- Capture output with streaming support
- Environment variable control

**Files to Create:**
- `pkg/mcp/command.go`
- `test/mcp_command.go`

**Dependencies:**
- Requires: Task 3.1 completed (MCP framework)

**Success Criteria:**
- Can execute safe commands
- Captures output correctly
- Timeouts work
- Dangerous commands blocked
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

## Phase 4: Intelligence & Learning

### Task 4.1: Method Cache & Retrieval

**Context:** As methods are proven through successful executions, they should be cached for reuse. Need efficient retrieval based on objective similarity and domain.

**Objective:** Implement method caching and retrieval system.

**Requirements:**
- Cache proven methods (success rate threshold)
- Retrieve by: domain, similarity to objective, recency
- Similarity matching: semantic (via embeddings) or structural
- Ranking: by success rate, recency, match quality
- Deprecation: mark superseded methods

**Key Design Decisions:**
- Query AmorphDB for methods by domain first
- Use embeddings for semantic similarity (via LLM service)
- Cache retrieval results temporarily (session)
- Return top N matches with confidence scores
- Consider success rate and recency in ranking

**Files to Create:**
- `pkg/core/method_cache.go`
- `test/method_cache.go`

**Dependencies:**
- Requires: Task 2.2 completed (Method model)
- Requires: Task 3.3 completed (LLM service for embeddings)

**Success Criteria:**
- Can cache and retrieve methods efficiently
- Similarity matching works
- Ranking produces sensible results
- All tests pass

**Estimated Lines:** ~250-300 lines total

---

### Task 4.2: LLM Router & Budget Manager

**Context:** Different tasks need different models (quality vs cost tradeoff). The router selects optimal models based on task requirements and budget constraints.

**Objective:** Implement intelligent LLM routing with budget management.

**Requirements:**
- Task assessment: complexity, tokens, quality needs
- Model selection: cheapest model meeting requirements
- Budget tracking: usage per day/week/month
- Cost estimation before execution
- Quality evaluation after execution
- Learning: track which models work best for which tasks

**Key Design Decisions:**
- Model capability matrix (simple table)
- Cost per token for each model
- Start conservative (quality over cost)
- Learn optimal choices from outcomes
- Budget alerts at thresholds (75%, 90%, 100%)

**Files to Create:**
- `pkg/llm/llm_router.go`
- `pkg/llm/budget_manager.go`
- `test/llm_router.go`
- `test/budget_manager.go`

**Dependencies:**
- Requires: Task 3.3 completed (LLM service)

**Success Criteria:**
- Routes tasks to appropriate models
- Tracks budget accurately
- Learns from outcomes
- Enforces budget limits
- All tests pass

**Estimated Lines:** ~300-400 lines total

---

### Task 4.3: User Context Learning

**Context:** The system learns about the user over time: preferences, patterns, values, constraints. This knowledge informs judgments and method selection. It's stored in AmorphDB and evolves temporally.

**Objective:** Implement user context learning and retrieval.

**Requirements:**
- Context categories: preferences, patterns, values, constraints, domain_expertise
- Learning sources: explicit statements, inferred from behavior, feedback
- Temporal evolution: track how preferences change
- Confidence scoring: how certain are we about this context
- Retrieval: relevant context for current objective

**Key Design Decisions:**
- Store as storage nodes type="UserContext"
- Categories are node properties, not separate types
- Confidence decreases over time (needs revalidation)
- Learn from: user corrections, feedback, repeated patterns
- Don't ask for context - infer and confirm when uncertain

**Files to Create:**
- `pkg/core/user_context.go`
- `test/user_context.go`

**Dependencies:**
- Requires: Task 1.3 completed (Storage query)

**Success Criteria:**
- Can store and retrieve user context
- Temporal evolution tracked
- Confidence scoring works
- Relevant context retrieval works
- All tests pass

**Estimated Lines:** ~250-300 lines total

---

### Task 4.4: Ethical Decision Framework

**Context:** The system makes ethical judgments based on the Prime Value (Mutual Freedom and Well-Being). It evaluates decisions, not by rules, but by impact on user freedom and well-being, plus system sustainability.

**Objective:** Implement ethical decision evaluation framework.

**Requirements:**
- Evaluate decisions for: user freedom impact, user well-being impact, system sustainability
- Not rule-based: contextual judgment using LLM
- Track decisions and outcomes for learning
- Flag decisions for user approval when uncertain
- Learn user's values from feedback

**Key Design Decisions:**
- Use LLM for ethical reasoning (structured prompt)
- Store decisions as storage nodes type="EthicalDecision"
- Link to objectives that triggered them
- User feedback refines judgment over time
- Always choose freedom over convenience when conflict

**Files to Create:**
- `pkg/core/ethical_framework.go`
- `test/ethical_framework.go`

**Dependencies:**
- Requires: Task 3.3 completed (LLM service)
- Requires: Task 4.3 completed (User context)

**Success Criteria:**
- Can evaluate decisions ethically
- Tracks decisions and outcomes
- Learns from feedback
- Appropriately flags for approval
- All tests pass

**Estimated Lines:** ~300-350 lines total

---

## Phase 5: User Interface & Integration

### Task 5.1: Command-Line Interface (Basic)

**Context:** Initial interface is a CLI for creating goals, submitting objectives, viewing status, and providing feedback. Simple and functional.

**Objective:** Build basic CLI for system interaction.

**Requirements:**
- Commands: create-goal, create-objective, list-goals, list-objectives, status, feedback
- Interactive mode: conversation-like interaction
- Status display: current objectives, recent completions
- Feedback: approve/reject decisions, rate outcomes
- Configuration: set preferences (budget limits, etc.)

**Key Design Decisions:**
- Use cobra or flag package for command parsing
- Store session state (current goal context)
- Pretty printing for status (tables)
- Keep it simple: text-based, no fancy TUI yet
- Config file for persistent settings

**Files to Create:**
- `cmd/studio/cli/main.go`
- `cmd/studio/cli/commands.go`
- `test/cli.go`

**Dependencies:**
- Requires: Phase 2 completed (Core components)

**Success Criteria:**
- All commands work correctly
- Interactive mode is usable
- Status display is clear
- Feedback mechanism works
- All tests pass

**Estimated Lines:** ~400-500 lines total

---

### Task 5.2: Background Agent Process

**Context:** The system should run in the background, monitoring for conditions that trigger objectives, executing scheduled tasks, and learning continuously.

**Objective:** Implement background daemon process.

**Requirements:**
- Run as background process (daemon)
- Monitor: scheduled objectives, conditional triggers, new goals
- Execution: run objectives autonomously when appropriate
- Interruption logic: when to notify user vs proceed
- Logging: activity log for debugging and review

**Key Design Decisions:**
- Use goroutines for concurrent monitoring
- Configurable polling intervals
- Interruption based on ethical framework + user context
- Activity log stored in file-based storage
- Graceful shutdown on SIGTERM

**Files to Create:**
- `cmd/agent/agent.go`
- `cmd/agent/scheduler.go`
- `test/daemon.go`

**Dependencies:**
- Requires: Phase 2, 4 completed (Core + intelligence)

**Success Criteria:**
- Runs as stable background process
- Executes objectives autonomously
- Interrupts appropriately
- Logs activity correctly
- All tests pass

**Estimated Lines:** ~300-400 lines total

---

### Task 5.3: Fyne GUI - Main Application Window

**Context:** Replace CLI-only interaction with a native GUI using Fyne. The main window provides navigation and houses all views.

**Objective:** Build the main Fyne application window with tab navigation.

**Requirements:**
- Main window with menu bar and tabs
- Tabs: Goals, Objectives, Methods, Status, Settings
- Application lifecycle management (startup, shutdown)
- Window size/position persistence
- Keyboard shortcuts for common actions

**Key Design Decisions:**
- Use Fyne's TabContainer for main navigation
- Menu bar for file operations and help
- Minimum window size enforced
- Save window preferences to config
- Native platform integration (system tray on supported platforms)

**Files to Create:**
- `cmd/studio/main.go`
- `pkg/ui/app.go`
- `pkg/ui/mainwindow.go`

**Dependencies:**
- Requires: Phase 2 completed (Core components)
- External: `fyne.io/fyne/v2`

**Success Criteria:**
- Application launches and displays correctly
- Tab navigation works
- Menu functions work
- Window preferences persist
- Graceful shutdown

**Estimated Lines:** ~250-350 lines total

---

### Task 5.4: Fyne GUI - Goals Management View

**Context:** The Goals tab displays goal hierarchy, allows CRUD operations, and shows status/priority visually.

**Objective:** Build the goals management interface in Fyne.

**Requirements:**
- Tree view of goal hierarchy
- Create/edit/archive dialogs
- Status and priority indicators
- Context menu (right-click) actions
- Search/filter capability

**Key Design Decisions:**
- Use Fyne's Tree widget for hierarchy
- Modal dialogs for forms (validation before save)
- Color-coded status indicators
- Refresh on data changes
- Keyboard navigation support

**Files to Create:**
- `pkg/ui/goals_view.go`
- `pkg/ui/goal_dialog.go`
- `pkg/ui/goals_view_test.go`

**Dependencies:**
- Requires: Task 5.3 completed (Main window)
- Requires: Task 2.1 completed (Goal model)

**Success Criteria:**
- Displays goal hierarchy correctly
- CRUD operations work
- Visual indicators are clear
- Search/filter functional
- All tests pass

**Estimated Lines:** ~400-500 lines total

---

### Task 5.5: Fyne GUI - Objectives Monitor View

**Context:** The Objectives tab shows active and completed objectives with status, progress, and results.

**Objective:** Build the objectives monitoring interface.

**Requirements:**
- List of objectives with status icons
- Create objective dialog with goal picker
- Progress indication for running objectives
- Results display when completed
- Real-time status updates

**Key Design Decisions:**
- Custom list item template with progress bar
- Auto-refresh via goroutine (updates every few seconds)
- Expandable detail view for each objective
- Filter dropdown (all, active, completed, failed)
- Click to view full results

**Files to Create:**
- `pkg/ui/objectives_view.go`
- `pkg/ui/objective_dialog.go`
- `pkg/ui/objectives_view_test.go`

**Dependencies:**
- Requires: Task 5.3 completed (Main window)
- Requires: Task 2.3 completed (Objective model)

**Success Criteria:**
- Lists objectives correctly
- Create dialog works
- Progress updates in real-time
- Results viewable
- All tests pass

**Estimated Lines:** ~400-500 lines total

---

### Task 5.6: Fyne GUI - Methods Library & Status Views

**Context:** The Methods tab shows learned methods with success rates and evolution. The Status tab provides system health dashboard.

**Objective:** Build methods library and status dashboard views.

**Requirements (Methods):**
- Table/list of methods with key metrics
- Detail pane showing method approach
- Version history visualization
- Search by domain/description

**Requirements (Status):**
- Current activity summary
- Recent completions/failures
- Budget usage charts
- System health indicators
- Quick action buttons

**Key Design Decisions:**
- Methods: Master-detail split layout
- Status: Card-based dashboard layout
- Use Fyne canvas for simple charts
- Real-time updates for status view
- Minimal, glanceable information

**Files to Create:**
- `pkg/ui/methods_view.go`
- `pkg/ui/status_view.go`
- `pkg/ui/charts.go`
- `pkg/ui/methods_view_test.go`
- `pkg/ui/status_view_test.go`

**Dependencies:**
- Requires: Task 5.3 completed (Main window)
- Requires: Task 2.2, 4.2 completed (Methods, Budget)

**Success Criteria:**
- Methods library displays correctly
- Status dashboard shows live data
- Charts are readable
- Search works
- All tests pass

**Estimated Lines:** ~500-600 lines total

---

## Phase 6: Deployment & Operations

### Task 6.1: Configuration Management

**Context:** System needs configuration for: database location, API keys, budget limits, allowed directories, etc. Use simple config file approach.

**Objective:** Implement configuration system.

**Requirements:**
- Config file: YAML or TOML format
- Settings: paths, API keys, budgets, permissions, preferences
- Environment override: env vars override config file
- Validation: ensure required settings present and valid
- Defaults: sensible defaults for optional settings

**Key Design Decisions:**
- Use TOML (simpler than YAML)
- Config file location: `~/.config/ai-work-studio/config.toml`
- API keys: prefer environment variables, fallback to config
- Validation on load: fail fast if invalid
- Example config provided in repo

**Files to Create:**
- `internal/config/manager.go`
- `internal/config/schema.go` (validation)
- `config/config.example.toml`
- `test/config.go`

**Dependencies:**
- None (foundational)

**Success Criteria:**
- Config loads correctly
- Validation works
- Environment override works
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

### Task 6.2: Logging & Monitoring

**Context:** Need comprehensive logging for debugging, auditing, and understanding system behavior. Different log levels for different components.

**Objective:** Implement structured logging system.

**Requirements:**
- Log levels: DEBUG, INFO, WARNING, ERROR
- Structured logs: JSON format for parsing
- Log destinations: file, console (configurable)
- Context: include relevant IDs (goal, objective, method)
- Rotation: automatic log file rotation

**Key Design Decisions:**
- Use structured logging (logrus or zap)
- Separate logs for: agent, mcp_services, amorphdb
- Include timestamps (ISO format), level, component, message, context
- Default: INFO to file, WARNING to console
- Configure via config file

**Files to Create:**
- `pkg/utils/logging.go`
- `test/logging.go`

**Dependencies:**
- None (foundational)

**Success Criteria:**
- Logging works across all components
- Structured format is parseable
- Rotation works
- Configurable via config
- All tests pass

**Estimated Lines:** ~150-200 lines total

---

### Task 6.3: Error Recovery & Resilience

**Context:** System should handle failures gracefully: API timeouts, disk full, corrupted data, etc. Need recovery strategies and resilience patterns.

**Objective:** Implement error recovery mechanisms.

**Requirements:**
- Retry logic: exponential backoff for transient failures
- Circuit breaker: stop retrying after repeated failures
- Graceful degradation: continue with reduced capability if possible
- State recovery: resume from last known good state
- Error reporting: notify user of critical failures

**Key Design Decisions:**
- Retry decorator for transient errors (API calls)
- Circuit breaker per service (track failure rate)
- Checkpointing: save state periodically during long executions
- AmorphDB persistence enables recovery
- Critical errors: log + notify user, don't crash silently

**Files to Create:**
- `pkg/utils/resilience.go`
- `test/resilience.go`

**Dependencies:**
- Requires: Task 6.2 completed (Logging)

**Success Criteria:**
- Retries work with backoff
- Circuit breaker prevents cascading failures
- State recovery works
- Errors reported appropriately
- All tests pass

**Estimated Lines:** ~200-250 lines total

---

### Task 6.4: Installation & Setup Script

**Context:** Make it easy to install and configure the system. Automated setup for first-time users.

**Objective:** Create installation and setup automation.

**Requirements:**
- Build system: Makefile or build script
- Configuration: interactive setup for first run
- Database: initialize storage directory structure
- Service setup: optional systemd service file for agent daemon
- Documentation: installation guide with platform-specific notes

**Key Design Decisions:**
- Use `go build` with proper flags (version, build time)
- Interactive setup on first run (walks through config)
- Create data directory structure automatically
- Prompts for: API keys, data directory, budget limits
- Cross-platform support (Windows, macOS, Linux)

**Files to Create:**
- `Makefile` or `build.sh`
- `install.sh` (Unix/Linux) and `install.ps1` (Windows)
- `docs/installation.md`
- `systemd/ai-work-studio.service` (Linux example)

**Dependencies:**
- None (foundational)

**Success Criteria:**
- Can build on all platforms
- Installation works smoothly
- Setup creates valid config
- Storage initializes correctly
- Documentation is clear

**Estimated Lines:** ~300-400 lines total (across scripts and docs)

---

## Phase 7: Testing & Documentation

### Task 7.1: Integration Test Suite

**Context:** Unit tests cover individual components, but need integration tests for end-to-end flows: creating goals, executing objectives, learning from results.

**Objective:** Build comprehensive integration test suite.

**Requirements:**
- Test scenarios: full objective execution, method learning, error recovery
- Mock external services: LLM APIs, web APIs
- Test data: sample goals, objectives, methods
- Assertions: verify correct state after each scenario
- Performance: tests should run in reasonable time

**Key Design Decisions:**
- Use Go.s testing package with table-driven tests
- Fixtures for common test data
- Mock LLM responses for determinism
- Test database separate from production
- Run tests in isolation (clean state each)

**Files to Create:**
- `test/integration_test.go`
- `test/learning_test.go`
- `test/recovery_test.go`
- `test/fixtures.go`
- `test/mocks.go`

**Dependencies:**
- Requires: All of Phases 1-4 completed

**Success Criteria:**
- All integration scenarios pass
- Tests are deterministic
- Run time < 2 minutes
- Good coverage of critical paths

**Estimated Lines:** ~500-600 lines total

---

### Task 7.2: API Documentation

**Context:** Document all public APIs for developers who want to extend or integrate with the system.

**Objective:** Create comprehensive API documentation using godoc.

**Requirements:**
- Package documentation: purpose, key types and functions
- Type documentation: methods, parameters, return types, examples
- MCP service documentation: operations, parameters, errors
- Storage documentation: schema, queries, best practices
- Code examples: common use cases
- Generate browsable docs with godoc

**Key Design Decisions:**
- Follow godoc conventions (complete sentences, starts with name)
- Package overview in `doc.go` files
- Examples as `Example*` test functions
- Keep examples minimal and focused
- Use `godoc` or `pkgsite` for viewing
- Markdown docs for high-level concepts

**Files to Create:**
- `pkg/core/doc.go`
- `pkg/storage/doc.go`
- `pkg/mcp/doc.go`
- `pkg/llm/doc.go`
- `docs/api/overview.md`
- `docs/tutorials/` (example code)

**Key Design Decisions:**
- Use docstrings (Google or NumPy style)
- Generate HTML docs with Sphinx or mkdocs
- Include: tutorials, API reference, architecture guide
- Keep examples minimal and focused
- Host docs locally or on GitHub Pages

**Files to Create:**
- `docs/api/core.md`
- `docs/api/amorphdb.md`
- `docs/api/mcp_services.md`
- `docs/api/router.md`
- `docs/tutorials/`

**Dependencies:**
- Requires: All code completed

**Success Criteria:**
- All public APIs documented
- Examples work correctly
- Documentation is navigable
- Generated docs look good

**Estimated Lines:** ~1000-1500 lines (markdown)

---

### Task 7.3: User Guide & Philosophy

**Context:** Help users understand what the system does, how to use it effectively, and the philosophy behind it.

**Objective:** Write comprehensive user guide.

**Requirements:**
- Getting started: installation, first goal, first objective
- Core concepts: goals, objectives, methods, learning
- Usage patterns: common workflows, best practices
- Configuration: tuning for your needs
- Philosophy: why it works this way, design principles
- Troubleshooting: common issues, debugging

**Key Design Decisions:**
- Start with quick start (5 minutes to first success)
- Explain concepts with examples
- Visual diagrams for complex flows
- FAQ section for common questions
- Link to philosophy document (original design doc)

**Files to Create:**
- `docs/user-guide/getting-started.md`
- `docs/user-guide/concepts.md`
- `docs/user-guide/workflows.md`
- `docs/user-guide/configuration.md`
- `docs/user-guide/troubleshooting.md`

**Dependencies:**
- Requires: System functional

**Success Criteria:**
- New users can get started in 5 minutes
- All concepts explained clearly
- Common workflows documented
- Troubleshooting helps resolve issues

**Estimated Lines:** ~800-1000 lines (markdown)

---

### Task 7.4: Performance Benchmarking

**Context:** Understand system performance characteristics: token usage, execution time, storage query speed, memory footprint.

**Objective:** Create benchmarking suite and performance baselines.

**Requirements:**
- Benchmarks: objective execution time, token usage, storage query speed, memory usage
- Test data: various complexity levels (simple, medium, complex objectives)
- Metrics: percentiles (p50, p95, p99), averages, max
- Reporting: generate performance report with visualizations
- Regression detection: alert if performance degrades

**Key Design Decisions:**
- Use Go's built-in benchmarking (`testing.B`)
- Mock LLM calls with realistic delays
- Measure token counts without actual API calls
- Profile with `pprof` for CPU/memory hotspots
- Store baselines in version control for comparison
- Run benchmarks with `go test -bench=. -benchmem`

**Files to Create:**
- `test/bench_test.go`
- `test/bench_report.go` (generates markdown report)
- `docs/performance.md` (baseline results)

**Dependencies:**
- Requires: All code completed

**Success Criteria:**
- Benchmarks run successfully
- Baselines established and documented
- Report is informative
- Can detect regressions
- Profile data useful for optimization

**Estimated Lines:** ~300-400 lines total

---

## Implementation Guidelines

### Task Session Protocol

**When starting a new task:**

1. **Read this section first** - Review development philosophy and conventions
2. **Read task description** - Understand objective and requirements
3. **Check dependencies** - Ensure prerequisite tasks are complete
4. **Review key design decisions** - Follow specified patterns
5. **Write tests first** - TDD approach when possible
6. **Implement incrementally** - Make it work, then refine
7. **Run tests** - Ensure all pass before considering complete
8. **Document** - Add/update docstrings and comments

**When task is complex:**

- Start with skeleton: classes, functions, interfaces
- Implement simplest case first
- Add complexity incrementally
- Refactor as you go (keep it simple)

**When stuck:**

- Review the design philosophy (simplicity over complexity)
- Check if you're adding unnecessary complexity
- Ask: "What's the simplest thing that could work?"
- Reference similar existing code
- Consider whether you need to escalate to CC (ask user for guidance)

### Code Quality Checklist

Before marking a task complete:

- [ ] All tests pass
- [ ] Code follows naming conventions
- [ ] Functions have docstrings
- [ ] Complex logic has comments
- [ ] No hardcoded values (use config)
- [ ] Error handling is present
- [ ] Logging is appropriate
- [ ] No obvious security issues
- [ ] Token usage is reasonable
- [ ] Passes simplicity test (can explain in 5 min)

### Token Budget Guidelines

Keep token usage reasonable per session:

- **Simple tasks (Task 1.x, 2.x):** ~5k-10k tokens
- **Medium tasks (Task 3.x, 4.x):** ~10k-20k tokens  
- **Complex tasks (Task 5.x+):** ~20k-40k tokens
- **Integration/docs:** ~20k-50k tokens

If a task balloons beyond these, consider:
- Breaking into smaller subtasks
- Simplifying the implementation
- Removing unnecessary complexity

### Dependency Management

**Add dependencies sparingly:**

Before adding a new Go module:
1. Can I do this with stdlib? (prefer yes)
2. Is this module well-maintained? (check GitHub stars, recent commits)
3. Does it have minimal dependencies itself? (check go.mod)
4. Will this still work in 2 years? (sustainability)
5. Is the license compatible? (check LICENSE file)

**Acceptable dependencies:**
- Core utilities: `github.com/google/uuid`
- GUI: `fyne.io/fyne/v2`
- LLM APIs: Official SDKs or direct HTTP with `net/http`
- HuggingFace: Evaluate options (may need to build HTTP client)
- Browser automation: `github.com/chromedp/chromedp` or `github.com/go-rod/rod`
- Logging: `github.com/sirupsen/logrus` or `go.uber.org/zap`
- CLI (if needed): `github.com/spf13/cobra`
- Config: `github.com/spf13/viper` or stdlib `encoding/json`

**Avoid:**
- Heavy frameworks (unless essential)
- Modules with many transitive dependencies
- Unmaintained packages (last commit >1 year ago)
- Vendor-specific SDKs when HTTP API works
- Anything requiring cgo unless absolutely necessary

**Managing go.mod:**
- Run `go mod tidy` regularly
- Keep dependencies up to date (but test after updates)
- Vendor if needed for reproducibility (`go mod vendor`)
- Document why each dependency exists (comment in imports)

---

## Success Criteria

**The system is ready when:**

1. ✅ All Phase 1-4 tasks complete (core functionality)
2. ✅ Integration tests pass
3. ✅ Can create goals and execute objectives end-to-end
4. ✅ Methods are learned and cached
5. ✅ Ethical framework makes reasonable judgments
6. ✅ Budget management works
7. ✅ CLI is usable
8. ✅ Background agent runs stably
9. ✅ Documentation is complete
10. ✅ Can run for a week without manual intervention

**Quality indicators:**

- Code is simple and understandable
- Tests are comprehensive
- Token usage is efficient
- Performance is acceptable
- User experience is smooth
- System learns and improves

---

## Notes for Claude Coder

**Context Window Management:**

Each task is designed to fit in a single session with room for:
- Reading this guide (~15k tokens)
- Task context and code (~20k-40k tokens)
- Test writing and iteration (~10k tokens)
- Buffer (~10k tokens)

**When you complete a task:**

1. Mark it complete in your session notes
2. Run all tests for that component
3. Note any deviations from the spec
4. List any dependencies for next task
5. Estimate actual lines of code written

**Communication style:**

- Be direct and clear
- Ask questions if requirements are ambiguous
- Suggest simplifications if design seems complex
- Flag potential issues early
- Explain your reasoning for design choices

**Remember:**

- Simplicity is the goal
- Judgment over rules
- Learn from practice
- One user, deep individualization
- Mutual freedom and well-being

Good luck! Build something beautiful through simplicity.

---

**Document Version:** 1.0  
**Created:** January 2026  
**Author:** Matthew Gries
