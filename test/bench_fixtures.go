package test

import (
	"os"
	"testing"

	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// NewBenchFixtures creates test fixtures for benchmarking.
// Similar to NewTestFixtures but works with *testing.B.
func NewBenchFixtures(b *testing.B) *TestFixtures {
	// Create temporary directory for test data
	testDir, err := os.MkdirTemp("", "ai-work-studio-bench-")
	if err != nil {
		b.Fatalf("Failed to create test directory: %v", err)
	}

	// Clean up on benchmark completion
	b.Cleanup(func() {
		os.RemoveAll(testDir)
	})

	// Initialize storage
	store, err := storage.NewStore(testDir)
	if err != nil {
		b.Fatalf("Failed to create test store: %v", err)
	}

	fixtures := &TestFixtures{
		TestDataDir:      testDir,
		Store:            store,
		GoalManager:      core.NewGoalManager(store),
		MethodManager:    core.NewMethodManager(store),
		ObjectiveManager: core.NewObjectiveManager(store),
	}

	// For benchmarks, we typically don't pre-populate with sample data
	// as it would skew the benchmark results

	return fixtures
}

