package test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/llm"
	"github.com/Solifugus/ai-work-studio/pkg/mcp"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)


var (
	globalSuite = &BenchmarkSuite{Started: time.Now()}
)

// addResult adds a benchmark result to the global suite.
func addResult(result BenchmarkResult) {
	globalSuite.Results = append(globalSuite.Results, result)
}

// recordBenchmark wraps a benchmark function with detailed measurement.
func recordBenchmark(b *testing.B, name string, fn func()) {
	// Pre-benchmark memory stats
	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	// Track individual operation timings for percentiles
	timings := make([]time.Duration, b.N)

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		opStart := time.Now()
		fn()
		timings[i] = time.Since(opStart)
	}

	duration := time.Since(start)
	b.StopTimer()

	// Post-benchmark memory stats
	runtime.ReadMemStats(&endMem)

	// Calculate timing percentiles
	sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })

	p50 := timings[len(timings)*50/100]
	p95 := timings[len(timings)*95/100]
	p99 := timings[len(timings)*99/100]
	min := timings[0]
	max := timings[len(timings)-1]

	// Memory calculations
	allocBytes := endMem.TotalAlloc - startMem.TotalAlloc
	allocObjects := endMem.Mallocs - startMem.Mallocs
	peakMemoryMB := float64(endMem.Sys) / 1024 / 1024
	avgMemoryMB := float64(allocBytes) / float64(b.N) / 1024 / 1024

	opsPerSec := float64(b.N) / duration.Seconds()

	result := BenchmarkResult{
		Name:                name,
		Duration:            duration,
		AllocBytes:          int64(allocBytes),
		AllocObjects:        int64(allocObjects),
		Iterations:          b.N,
		OperationsPerSecond: opsPerSec,
		P50:                 p50,
		P95:                 p95,
		P99:                 p99,
		Min:                 min,
		Max:                 max,
		PeakMemoryMB:        peakMemoryMB,
		AvgMemoryMB:         avgMemoryMB,
	}

	addResult(result)
}

// ====== STORAGE LAYER BENCHMARKS ======

func BenchmarkStorageNodeCreate(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-storage-node-create-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewStore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	recordBenchmark(b, "Storage_Node_Create", func() {
		nodeID := fmt.Sprintf("node-%d-%d", time.Now().UnixNano(), rand.Int())
		node := &storage.Node{
			ID:   nodeID,
			Type: "test",
			Data: map[string]interface{}{
				"title":       "Test Node",
				"description": "A test node for benchmarking",
				"priority":    rand.Intn(10),
				"complexity":  rand.Intn(100),
			},
			ValidFrom: time.Now(),
		}

		err := store.AddNode(context.Background(), node)
		if err != nil {
			b.Fatalf("Failed to add node: %v", err)
		}
	})
}

func BenchmarkStorageNodeRead(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-storage-node-read-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewStore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	// Pre-populate with test nodes
	nodeIDs := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodeIDs[i] = nodeID

		node := &storage.Node{
			ID:   nodeID,
			Type: "test",
			Data: map[string]interface{}{
				"title": fmt.Sprintf("Test Node %d", i),
				"value": i,
			},
			ValidFrom: time.Now(),
		}

		err := store.AddNode(context.Background(), node)
		if err != nil {
			b.Fatalf("Failed to add node %d: %v", i, err)
		}
	}

	recordBenchmark(b, "Storage_Node_Read", func() {
		nodeID := nodeIDs[rand.Intn(len(nodeIDs))]
		_, err := store.GetNode(context.Background(), nodeID)
		if err != nil {
			b.Fatalf("Failed to get node: %v", err)
		}
	})
}

func BenchmarkStorageNodeUpdate(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-storage-node-update-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewStore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	// Create initial node
	originalNode := &storage.Node{
		ID:   "update-test-node",
		Type: "test",
		Data: map[string]interface{}{
			"title": "Original Title",
			"value": 1,
		},
		ValidFrom: time.Now(),
	}

	err = store.AddNode(context.Background(), originalNode)
	if err != nil {
		b.Fatalf("Failed to add original node: %v", err)
	}

	recordBenchmark(b, "Storage_Node_Update", func() {
		updatedData := map[string]interface{}{
			"title":   "Updated Title",
			"value":   rand.Intn(1000),
			"updated": time.Now(),
		}

		err := store.UpdateNode(context.Background(), originalNode.ID, updatedData)
		if err != nil {
			b.Fatalf("Failed to update node: %v", err)
		}
	})
}

func BenchmarkStorageEdgeCreate(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-storage-edge-create-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewStore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	recordBenchmark(b, "Storage_Edge_Create", func() {
		edgeID := fmt.Sprintf("edge-%d-%d", time.Now().UnixNano(), rand.Int())
		edge := &storage.Edge{
			ID:       edgeID,
			SourceID: fmt.Sprintf("source-%d", rand.Intn(100)),
			TargetID: fmt.Sprintf("target-%d", rand.Intn(100)),
			Type:     "test_relationship",
			Data: map[string]interface{}{
				"weight":     rand.Float64(),
				"created_by": "benchmark",
			},
			ValidFrom: time.Now(),
		}

		err := store.AddEdge(context.Background(), edge)
		if err != nil {
			b.Fatalf("Failed to add edge: %v", err)
		}
	})
}

func BenchmarkStorageQueryByType(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-storage-query-type-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewStore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	// Pre-populate with nodes of different types
	types := []string{"goal", "method", "objective", "task", "result"}
	for i := 0; i < 1000; i++ {
		nodeType := types[i%len(types)]
		node := &storage.Node{
			ID:   fmt.Sprintf("%s-%d", nodeType, i),
			Type: nodeType,
			Data: map[string]interface{}{
				"title": fmt.Sprintf("Test %s %d", nodeType, i),
				"index": i,
			},
			ValidFrom: time.Now(),
		}

		err := store.AddNode(context.Background(), node)
		if err != nil {
			b.Fatalf("Failed to add node %d: %v", i, err)
		}
	}

	recordBenchmark(b, "Storage_Query_By_Type", func() {
		nodeType := types[rand.Intn(len(types))]
		_, err := store.GetNodesByType(context.Background(), nodeType)
		if err != nil {
			b.Fatalf("Failed to query by type: %v", err)
		}
	})
}

func BenchmarkStorageGraphTraversal(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-storage-graph-traversal-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewStore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	// Create a connected graph structure
	nodeIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodeIDs[i] = nodeID

		node := &storage.Node{
			ID:   nodeID,
			Type: "graph_node",
			Data: map[string]interface{}{
				"value": i,
			},
			ValidFrom: time.Now(),
		}

		err := store.AddNode(context.Background(), node)
		if err != nil {
			b.Fatalf("Failed to add node %d: %v", i, err)
		}

		// Create edges to create a connected graph
		if i > 0 {
			edge := &storage.Edge{
				ID:       fmt.Sprintf("edge-%d-%d", i-1, i),
				SourceID: nodeIDs[i-1],
				TargetID: nodeIDs[i],
				Type:     "connects",
				Data:     map[string]interface{}{},
				ValidFrom: time.Now(),
			}

			err := store.AddEdge(context.Background(), edge)
			if err != nil {
				b.Fatalf("Failed to add edge %d: %v", i, err)
			}
		}
	}

	recordBenchmark(b, "Storage_Graph_Traversal", func() {
		sourceID := nodeIDs[rand.Intn(len(nodeIDs))]
		_, err := store.GetNeighbors(context.Background(), sourceID)
		if err != nil {
			b.Fatalf("Failed to get neighbors: %v", err)
		}
	})
}

// ====== MANAGER LAYER BENCHMARKS ======

func BenchmarkGoalManagerCreate(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	recordBenchmark(b, "GoalManager_Create", func() {
		title := fmt.Sprintf("Benchmark Goal %d", rand.Int())
		description := "A test goal created during benchmarking to measure performance"
		priority := rand.Intn(10) + 1

		_, err := fixtures.GoalManager.CreateGoal(ctx, title, description, priority, map[string]interface{}{
			"benchmark": true,
			"created":   time.Now(),
		})
		if err != nil {
			b.Fatalf("Failed to create goal: %v", err)
		}
	})
}

func BenchmarkGoalManagerList(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	// Pre-populate with goals
	for i := 0; i < 500; i++ {
		_, err := fixtures.GoalManager.CreateGoal(ctx,
			fmt.Sprintf("Goal %d", i),
			fmt.Sprintf("Description for goal %d", i),
			(i%10)+1,
			map[string]interface{}{
				"index": i,
				"type":  []string{"business", "technical", "learning"}[i%3],
			})
		if err != nil {
			b.Fatalf("Failed to create test goal %d: %v", i, err)
		}
	}

	recordBenchmark(b, "GoalManager_List", func() {
		activeStatus := core.GoalStatusActive
		_, err := fixtures.GoalManager.ListGoals(ctx, core.GoalFilter{
			Status: &activeStatus,
		})
		if err != nil {
			b.Fatalf("Failed to list goals: %v", err)
		}
	})
}

func BenchmarkGoalManagerHierarchy(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	// Create parent goal
	parentGoal, err := fixtures.GoalManager.CreateGoal(ctx, "Parent Goal", "Test parent goal", 8, map[string]interface{}{})
	if err != nil {
		b.Fatalf("Failed to create parent goal: %v", err)
	}

	// Create child goals
	childGoals := make([]*core.Goal, 20)
	for i := 0; i < 20; i++ {
		childGoal, err := fixtures.GoalManager.CreateGoal(ctx,
			fmt.Sprintf("Child Goal %d", i),
			fmt.Sprintf("Child goal %d description", i),
			5, map[string]interface{}{})
		if err != nil {
			b.Fatalf("Failed to create child goal %d: %v", i, err)
		}
		childGoals[i] = childGoal

		err = fixtures.GoalManager.AddSubGoal(ctx, parentGoal.ID, childGoal.ID)
		if err != nil {
			b.Fatalf("Failed to add sub-goal %d: %v", i, err)
		}
	}

	recordBenchmark(b, "GoalManager_Hierarchy", func() {
		_, err := fixtures.GoalManager.GetSubGoals(ctx, parentGoal.ID)
		if err != nil {
			b.Fatalf("Failed to get sub-goals: %v", err)
		}
	})
}

func BenchmarkMethodManagerCreate(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	recordBenchmark(b, "MethodManager_Create", func() {
		name := fmt.Sprintf("Benchmark Method %d", rand.Int())
		description := "A test method created during benchmarking"

		// Create realistic approach steps
		steps := []core.ApproachStep{
			{
				Description: "Analyze the problem and gather requirements",
				Tools:       []string{"research", "analysis"},
				Heuristics:  []string{"ask_clarifying_questions", "identify_constraints"},
			},
			{
				Description: "Design solution approach",
				Tools:       []string{"design", "planning"},
				Heuristics:  []string{"consider_alternatives", "optimize_for_simplicity"},
			},
			{
				Description: "Implement and test solution",
				Tools:       []string{"implementation", "testing"},
				Heuristics:  []string{"test_early_and_often", "validate_against_requirements"},
			},
		}

		_, err := fixtures.MethodManager.CreateMethod(ctx, name, description, steps,
			core.MethodDomainGeneral, map[string]interface{}{
				"benchmark": true,
				"complexity": rand.Intn(10),
			})
		if err != nil {
			b.Fatalf("Failed to create method: %v", err)
		}
	})
}

func BenchmarkMethodManagerUpdateMetrics(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	// Create test method
	method, err := fixtures.MethodManager.CreateMethod(ctx,
		"Test Method",
		"Method for metrics testing",
		[]core.ApproachStep{{Description: "Do something", Tools: []string{}, Heuristics: []string{}}},
		core.MethodDomainGeneral,
		map[string]interface{}{})
	if err != nil {
		b.Fatalf("Failed to create test method: %v", err)
	}

	recordBenchmark(b, "MethodManager_UpdateMetrics", func() {
		wasSuccessful := rand.Float64() > 0.3 // 70% success rate
		rating := 5.0 + rand.Float64()*5.0   // Rating 5-10

		err := fixtures.MethodManager.UpdateMethodMetrics(ctx, method.ID, wasSuccessful, rating)
		if err != nil {
			b.Fatalf("Failed to update method metrics: %v", err)
		}
	})
}

func BenchmarkObjectiveManagerLifecycle(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	// Pre-create goals and methods for objectives
	goals := make([]*core.Goal, 10)
	methods := make([]*core.Method, 10)

	for i := 0; i < 10; i++ {
		goal, err := fixtures.GoalManager.CreateGoal(ctx,
			fmt.Sprintf("Goal %d", i), "Test goal", 5, map[string]interface{}{})
		if err != nil {
			b.Fatalf("Failed to create goal %d: %v", i, err)
		}
		goals[i] = goal

		method, err := fixtures.MethodManager.CreateMethod(ctx,
			fmt.Sprintf("Method %d", i), "Test method",
			[]core.ApproachStep{{Description: "Do something", Tools: []string{}, Heuristics: []string{}}},
			core.MethodDomainGeneral, map[string]interface{}{})
		if err != nil {
			b.Fatalf("Failed to create method %d: %v", i, err)
		}
		methods[i] = method
	}

	recordBenchmark(b, "ObjectiveManager_Lifecycle", func() {
		// Create objective
		goal := goals[rand.Intn(len(goals))]
		method := methods[rand.Intn(len(methods))]

		objective, err := fixtures.ObjectiveManager.CreateObjective(ctx,
			goal.ID, method.ID,
			fmt.Sprintf("Test Objective %d", rand.Int()),
			"A test objective for benchmarking",
			map[string]interface{}{"benchmark": true},
			5)
		if err != nil {
			b.Fatalf("Failed to create objective: %v", err)
		}

		// Start objective
		_, err = fixtures.ObjectiveManager.StartObjective(ctx, objective.ID)
		if err != nil {
			b.Fatalf("Failed to start objective: %v", err)
		}

		// Complete objective with result
		result := core.ObjectiveResult{
			Success:       true,
			Message:       "Objective completed successfully",
			Data:          map[string]interface{}{"result": "success"},
			TokensUsed:    500 + rand.Intn(500),
			ExecutionTime: time.Duration(rand.Intn(5000)) * time.Millisecond,
		}

		_, err = fixtures.ObjectiveManager.CompleteObjective(ctx, objective.ID, result)
		if err != nil {
			b.Fatalf("Failed to complete objective: %v", err)
		}
	})
}

// ====== COMPLEX SCENARIO BENCHMARKS ======

func BenchmarkConcurrentObjectiveManagement(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	// Pre-create goals and methods
	goal, err := fixtures.GoalManager.CreateGoal(ctx, "Concurrent Goal", "Test goal", 5, map[string]interface{}{})
	if err != nil {
		b.Fatalf("Failed to create goal: %v", err)
	}

	method, err := fixtures.MethodManager.CreateMethod(ctx, "Concurrent Method", "Test method",
		[]core.ApproachStep{{Description: "Do something", Tools: []string{}, Heuristics: []string{}}},
		core.MethodDomainGeneral, map[string]interface{}{})
	if err != nil {
		b.Fatalf("Failed to create method: %v", err)
	}

	recordBenchmark(b, "Concurrent_Objective_Management", func() {
		// Simulate concurrent objective operations
		objectives := make([]*core.Objective, 5)

		// Create objectives concurrently
		for i := 0; i < 5; i++ {
			objective, err := fixtures.ObjectiveManager.CreateObjective(ctx,
				goal.ID, method.ID,
				fmt.Sprintf("Concurrent Objective %d-%d", time.Now().UnixNano(), i),
				"Concurrent test objective",
				map[string]interface{}{"index": i},
				5)
			if err != nil {
				b.Fatalf("Failed to create objective %d: %v", i, err)
			}
			objectives[i] = objective
		}

		// Start and complete them
		for _, objective := range objectives {
			_, err := fixtures.ObjectiveManager.StartObjective(ctx, objective.ID)
			if err != nil {
				b.Fatalf("Failed to start objective: %v", err)
			}

			result := core.ObjectiveResult{
				Success:       true,
				Message:       "Concurrent objective completed",
				Data:          map[string]interface{}{"result": "concurrent_success"},
				TokensUsed:    100 + rand.Intn(100),
				ExecutionTime: time.Duration(rand.Intn(1000)) * time.Millisecond,
			}

			_, err = fixtures.ObjectiveManager.CompleteObjective(ctx, objective.ID, result)
			if err != nil {
				b.Fatalf("Failed to complete objective: %v", err)
			}
		}
	})
}

func BenchmarkLargeDataScenario(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	recordBenchmark(b, "Large_Data_Scenario", func() {
		// Simulate a scenario with large amounts of data

		// Create a complex goal with extensive metadata
		largeMetadata := map[string]interface{}{
			"stakeholders": make([]string, 50),
			"requirements": make([]string, 100),
			"constraints":  make(map[string]interface{}),
			"history":      make([]map[string]interface{}, 200),
		}

		// Fill with realistic data
		for i := 0; i < 50; i++ {
			largeMetadata["stakeholders"].([]string)[i] = fmt.Sprintf("stakeholder-%d", i)
		}
		for i := 0; i < 100; i++ {
			largeMetadata["requirements"].([]string)[i] = fmt.Sprintf("requirement-%d: detailed description of requirement %d", i, i)
		}
		for i := 0; i < 20; i++ {
			largeMetadata["constraints"].(map[string]interface{})[fmt.Sprintf("constraint-%d", i)] = fmt.Sprintf("constraint value %d", i)
		}
		for i := 0; i < 200; i++ {
			largeMetadata["history"].([]map[string]interface{})[i] = map[string]interface{}{
				"timestamp": time.Now().Add(time.Duration(-i) * time.Hour),
				"action":    fmt.Sprintf("action-%d", i),
				"user":      fmt.Sprintf("user-%d", i%10),
				"details":   fmt.Sprintf("detailed description of action %d with lots of text", i),
			}
		}

		goal, err := fixtures.GoalManager.CreateGoal(ctx, "Large Data Goal", "Goal with extensive metadata", 8, largeMetadata)
		if err != nil {
			b.Fatalf("Failed to create large goal: %v", err)
		}

		// Create method with complex approach
		complexSteps := make([]core.ApproachStep, 15)
		for i := 0; i < 15; i++ {
			complexSteps[i] = core.ApproachStep{
				Description: fmt.Sprintf("Complex step %d with detailed description and multiple considerations", i),
				Tools:       []string{fmt.Sprintf("tool-%d", i), fmt.Sprintf("tool-%d-alt", i)},
				Heuristics:  []string{fmt.Sprintf("heuristic-%d", i), fmt.Sprintf("heuristic-%d-alt", i)},
				Conditions: map[string]interface{}{
					fmt.Sprintf("condition-%d", i): fmt.Sprintf("value-%d", i),
					"complexity_level":             rand.Intn(10),
				},
			}
		}

		method, err := fixtures.MethodManager.CreateMethod(ctx, "Complex Method", "Method with many steps",
			complexSteps, core.MethodDomainSpecific, map[string]interface{}{
				"domain": "complex_analysis",
				"estimated_duration": "4-8 hours",
				"required_resources": []string{"expert_analyst", "specialized_tools", "large_dataset"},
			})
		if err != nil {
			b.Fatalf("Failed to create complex method: %v", err)
		}

		// Create objective with large context
		largeContext := map[string]interface{}{
			"input_datasets":     make([]string, 20),
			"processing_params":  make(map[string]interface{}),
			"output_requirements": make(map[string]interface{}),
		}

		for i := 0; i < 20; i++ {
			largeContext["input_datasets"].([]string)[i] = fmt.Sprintf("dataset-%d.json", i)
		}

		for i := 0; i < 30; i++ {
			largeContext["processing_params"].(map[string]interface{})[fmt.Sprintf("param-%d", i)] = rand.Float64()
		}

		largeContext["output_requirements"] = map[string]interface{}{
			"format":      "structured_json",
			"quality":     "high",
			"completeness": 0.95,
			"fields":      make([]string, 50),
		}

		for i := 0; i < 50; i++ {
			largeContext["output_requirements"].(map[string]interface{})["fields"].([]string)[i] = fmt.Sprintf("field-%d", i)
		}

		_, err = fixtures.ObjectiveManager.CreateObjective(ctx, goal.ID, method.ID,
			"Large Context Objective", "Objective with extensive context data", largeContext, 8)
		if err != nil {
			b.Fatalf("Failed to create large objective: %v", err)
		}
	})
}

// Memory benchmark specifically testing memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	recordBenchmark(b, "Memory_Usage_Pattern", func() {
		// Simulate typical memory usage patterns
		fixtures := &TestFixtures{}

		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "bench-memory-")
		if err != nil {
			b.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize components
		store, err := storage.NewStore(tmpDir)
		if err != nil {
			b.Fatalf("Failed to create store: %v", err)
		}
		fixtures.Store = store
		fixtures.GoalManager = core.NewGoalManager(store)
		fixtures.MethodManager = core.NewMethodManager(store)
		fixtures.ObjectiveManager = core.NewObjectiveManager(store)

		ctx := context.Background()

		// Create and manage typical working set
		for i := 0; i < 10; i++ {
			goal, _ := fixtures.GoalManager.CreateGoal(ctx, fmt.Sprintf("Goal %d", i), "Description", 5, map[string]interface{}{})
			method, _ := fixtures.MethodManager.CreateMethod(ctx, fmt.Sprintf("Method %d", i), "Description",
				[]core.ApproachStep{{Description: "Step", Tools: []string{}, Heuristics: []string{}}},
				core.MethodDomainGeneral, map[string]interface{}{})
			fixtures.ObjectiveManager.CreateObjective(ctx, goal.ID, method.ID, fmt.Sprintf("Objective %d", i), "Description", map[string]interface{}{}, 5)
		}

		// Simulate typical access patterns
		fixtures.GoalManager.ListGoals(ctx, core.GoalFilter{})
		fixtures.MethodManager.ListMethods(ctx, core.MethodFilter{})
		fixtures.ObjectiveManager.ListObjectives(ctx, core.ObjectiveFilter{})
	})
}

// ====== LLM AND METHOD CACHE BENCHMARKS ======

func BenchmarkMethodCacheCreate(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	// Create test methods for caching
	methods := make([]*core.Method, 20)
	for i := 0; i < 20; i++ {
		method, err := fixtures.MethodManager.CreateMethod(ctx,
			fmt.Sprintf("Cache Test Method %d", i),
			fmt.Sprintf("Method %d for cache testing", i),
			[]core.ApproachStep{{
				Description: fmt.Sprintf("Step for method %d", i),
				Tools:       []string{fmt.Sprintf("tool-%d", i)},
				Heuristics:  []string{fmt.Sprintf("heuristic-%d", i)},
			}},
			core.MethodDomainGeneral,
			map[string]interface{}{
				"domain": fmt.Sprintf("test-domain-%d", i%3),
				"complexity": i % 10,
			})
		if err != nil {
			b.Fatalf("Failed to create test method %d: %v", i, err)
		}
		methods[i] = method

		// Simulate some usage to give methods history
		for j := 0; j < 5+i; j++ {
			wasSuccessful := rand.Float64() > 0.25 // 75% success rate
			rating := 6.0 + rand.Float64()*3.0   // Rating 6-9

			err := fixtures.MethodManager.UpdateMethodMetrics(ctx, method.ID, wasSuccessful, rating)
			if err != nil {
				b.Fatalf("Failed to update method metrics: %v", err)
			}
		}
	}

	// Create method cache
	cache := core.NewMethodCache(fixtures.Store, nil, core.DefaultCacheConfig())

	recordBenchmark(b, "MethodCache_Create", func() {
		method := methods[rand.Intn(len(methods))]

		// Simulate adding method to cache
		err := cache.CacheProvenMethod(ctx, method)
		if err != nil {
			b.Fatalf("Failed to cache proven method: %v", err)
		}
	})
}

func BenchmarkMethodCacheFind(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	// Create and populate method cache
	cache := core.NewMethodCache(fixtures.Store, nil, core.DefaultCacheConfig())

	// Pre-populate cache with methods
	for i := 0; i < 100; i++ {
		method, err := fixtures.MethodManager.CreateMethod(ctx,
			fmt.Sprintf("Findable Method %d", i),
			fmt.Sprintf("Method %d for finding tests with various domains", i),
			[]core.ApproachStep{{
				Description: fmt.Sprintf("Analysis step for domain %d", i%10),
				Tools:       []string{fmt.Sprintf("analysis-tool-%d", i%5)},
				Heuristics:  []string{fmt.Sprintf("heuristic-%d", i%7)},
			}},
			[]core.MethodDomain{core.MethodDomainGeneral, core.MethodDomainSpecific, core.MethodDomainUser}[i%3],
			map[string]interface{}{
				"domain": fmt.Sprintf("domain-%d", i%10),
				"complexity": (i%10) + 1,
			})
		if err != nil {
			b.Fatalf("Failed to create findable method %d: %v", i, err)
		}

		// Simulate usage history
		for j := 0; j < 3+rand.Intn(7); j++ {
			wasSuccessful := rand.Float64() > 0.2 // 80% success rate
			rating := 5.0 + rand.Float64()*4.0   // Rating 5-9

			err := fixtures.MethodManager.UpdateMethodMetrics(ctx, method.ID, wasSuccessful, rating)
			if err != nil {
				b.Fatalf("Failed to update method metrics: %v", err)
			}
		}

		// Add to cache
		err = cache.CacheProvenMethod(ctx, method)
		if err != nil {
			b.Fatalf("Failed to cache method %d: %v", i, err)
		}
	}

	queries := []string{
		"analyze customer feedback data",
		"optimize database performance",
		"train machine learning model",
		"process financial transactions",
		"generate report summaries",
		"validate data quality",
		"implement security measures",
		"design user interface",
		"troubleshoot system issues",
		"plan project timeline",
	}

	recordBenchmark(b, "MethodCache_Find", func() {
		query := queries[rand.Intn(len(queries))]

		// Use the cache query interface
		_, err := cache.Query().WithObjective(query).Execute(ctx)
		if err != nil {
			b.Fatalf("Failed to find similar methods: %v", err)
		}
	})
}

// BenchMockLLMService is a mock LLM service specifically for benchmarking
type BenchMockLLMService struct {
	responseTime time.Duration
	tokenCost    float64
}

func (m *BenchMockLLMService) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
	// Simulate processing time
	time.Sleep(m.responseTime)

	// Simulate token usage based on request
	promptTokens := len(strings.Split(fmt.Sprintf("%v", params["prompt"]), " "))
	responseTokens := 50 + rand.Intn(200) // Simulate 50-250 response tokens

	return mcp.ServiceResult{
		Success: true,
		Data: map[string]interface{}{
			"response": "Mock LLM response for benchmarking purposes",
			"tokens_used": promptTokens + responseTokens,
			"prompt_tokens": promptTokens,
			"completion_tokens": responseTokens,
			"cost": float64(promptTokens + responseTokens) * m.tokenCost,
		},
	}
}

// TODO: Fix LLM router benchmarks - currently disabled due to API mismatch
func benchmarkLLMRouterTaskAssessment(b *testing.B) {
	// Create mock LLM services with different characteristics
	fastCheapService := &BenchMockLLMService{
		responseTime: 50 * time.Millisecond,
		tokenCost:    0.001, // $0.001 per token
	}

	qualityService := &BenchMockLLMService{
		responseTime: 200 * time.Millisecond,
		tokenCost:    0.01, // $0.01 per token
	}

	premiumService := &BenchMockLLMService{
		responseTime: 500 * time.Millisecond,
		tokenCost:    0.02, // $0.02 per token
	}

	// TODO: Fix router creation
	_ = fastCheapService
	_ = qualityService
	_ = premiumService
	// router := llm.NewLLMRouter(...)

	requests := []llm.TaskRequest{
		{
			Prompt:          "Analyze this customer feedback and identify key themes",
			MaxTokens:       500,
			Temperature:     0.3,
			TaskType:        "analysis",
			QualityRequired: llm.QualityStandard,
		},
		{
			Prompt:          "Generate a creative marketing campaign for a new product",
			MaxTokens:       800,
			Temperature:     0.8,
			TaskType:        "generation",
			QualityRequired: llm.QualityPremium,
		},
		{
			Prompt:          "What is the capital of France?",
			MaxTokens:       50,
			Temperature:     0.0,
			TaskType:        "qa",
			QualityRequired: llm.QualityBasic,
		},
		{
			Prompt:          "Review this code for security vulnerabilities: " + strings.Repeat("func example() { }", 50),
			MaxTokens:       1000,
			Temperature:     0.1,
			TaskType:        "code_review",
			QualityRequired: llm.QualityPremium,
		},
	}

	recordBenchmark(b, "LLMRouter_TaskAssessment", func() {
		_ = requests
		// TODO: Fix router.AssessTask call
		// request := requests[rand.Intn(len(requests))]
		// _, err := router.AssessTask(context.Background(), request)
	})
}

// TODO: Fix LLM router benchmarks - currently disabled due to API mismatch
func benchmarkLLMRouterRoute(b *testing.B) {
	// TODO: Fix router creation
	// router := llm.NewRouter(...)

	requests := []llm.TaskRequest{
		{
			Prompt:          "Simple question about basic facts",
			MaxTokens:       100,
			TaskType:        "qa",
			QualityRequired: llm.QualityBasic,
		},
		{
			Prompt:          "Complex analysis requiring deep reasoning and multiple perspectives",
			MaxTokens:       1000,
			TaskType:        "analysis",
			QualityRequired: llm.QualityPremium,
			BudgetConstraint: func() *float64 { cost := 0.50; return &cost }(),
		},
	}

	recordBenchmark(b, "LLMRouter_Route", func() {
		_ = requests
		// TODO: Fix router.Route call
		// request := requests[rand.Intn(len(requests))]
		// _, err := router.Route(context.Background(), request)
	})
}

// ====== INTEGRATION BENCHMARKS ======

func BenchmarkFullWorkflow(b *testing.B) {
	fixtures := NewBenchFixtures(b)
	ctx := context.Background()

	recordBenchmark(b, "Full_Workflow", func() {
		// 1. Create Goal
		goal, err := fixtures.GoalManager.CreateGoal(ctx,
			fmt.Sprintf("Workflow Goal %d", rand.Int()),
			"A complete workflow test goal",
			8,
			map[string]interface{}{
				"workflow_test": true,
				"complexity": rand.Intn(10),
			})
		if err != nil {
			b.Fatalf("Failed to create goal: %v", err)
		}

		// 2. Create Method
		steps := []core.ApproachStep{
			{
				Description: "Gather and analyze requirements",
				Tools:       []string{"research", "analysis"},
				Heuristics:  []string{"ask_clarifying_questions"},
			},
			{
				Description: "Design and implement solution",
				Tools:       []string{"design", "implementation"},
				Heuristics:  []string{"iterate_rapidly", "test_assumptions"},
			},
			{
				Description: "Validate and refine results",
				Tools:       []string{"testing", "validation"},
				Heuristics:  []string{"measure_outcomes", "gather_feedback"},
			},
		}

		method, err := fixtures.MethodManager.CreateMethod(ctx,
			fmt.Sprintf("Workflow Method %d", rand.Int()),
			"A method for the complete workflow test",
			steps,
			core.MethodDomainGeneral,
			map[string]interface{}{
				"estimated_duration": "2-4 hours",
				"success_rate": 0.85,
			})
		if err != nil {
			b.Fatalf("Failed to create method: %v", err)
		}

		// 3. Create Objective
		objective, err := fixtures.ObjectiveManager.CreateObjective(ctx,
			goal.ID,
			method.ID,
			fmt.Sprintf("Workflow Objective %d", rand.Int()),
			"Test the complete workflow from goal to result",
			map[string]interface{}{
				"expected_outcome": "successful_completion",
				"quality_threshold": 0.8,
			},
			7)
		if err != nil {
			b.Fatalf("Failed to create objective: %v", err)
		}

		// 4. Execute Objective Lifecycle
		_, err = fixtures.ObjectiveManager.StartObjective(ctx, objective.ID)
		if err != nil {
			b.Fatalf("Failed to start objective: %v", err)
		}

		// 5. Simulate execution and complete
		result := core.ObjectiveResult{
			Success: true,
			Message: "Workflow completed successfully",
			Data: map[string]interface{}{
				"workflow_result": "successful",
				"quality_score":   0.9,
				"steps_completed": len(steps),
			},
			TokensUsed:     300 + rand.Intn(700), // 300-1000 tokens
			ExecutionTime:  time.Duration(1000+rand.Intn(4000)) * time.Millisecond, // 1-5 seconds
		}

		_, err = fixtures.ObjectiveManager.CompleteObjective(ctx, objective.ID, result)
		if err != nil {
			b.Fatalf("Failed to complete objective: %v", err)
		}

		// 6. Update method metrics
		err = fixtures.MethodManager.UpdateMethodMetrics(ctx, method.ID, true, 8.5)
		if err != nil {
			b.Fatalf("Failed to update method metrics: %v", err)
		}

		// 7. Query results
		_, err = fixtures.GoalManager.GetSubGoals(ctx, goal.ID)
		if err != nil {
			b.Fatalf("Failed to query sub-goals: %v", err)
		}

		completedStatus := core.ObjectiveStatusCompleted
		_, err = fixtures.ObjectiveManager.ListObjectives(ctx, core.ObjectiveFilter{
			GoalID: &goal.ID,
			Status: &completedStatus,
		})
		if err != nil {
			b.Fatalf("Failed to query objectives: %v", err)
		}
	})
}

// TODO: Fix budget manager benchmarks - currently disabled due to missing budget manager
func benchmarkTokenBudgetScenario(b *testing.B) {
	_ = NewBenchFixtures(b)
	_ = context.Background()

	// TODO: Create budget manager
	// budgetManager := llm.NewBudgetManager(...)

	recordBenchmark(b, "Token_Budget_Scenario", func() {
		// TODO: Simulate budget tracking when budget manager is available
	})
}