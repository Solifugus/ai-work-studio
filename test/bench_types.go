package test

import (
	"time"
)

// BenchmarkResult stores timing and memory statistics for performance analysis.
type BenchmarkResult struct {
	Name           string
	Duration       time.Duration
	AllocBytes     int64
	AllocObjects   int64
	Iterations     int
	OperationsPerSecond float64

	// Detailed timing percentiles
	P50, P95, P99  time.Duration
	Min, Max       time.Duration

	// Memory usage
	PeakMemoryMB   float64
	AvgMemoryMB    float64
}

// BenchmarkSuite manages the collection of all benchmark results.
type BenchmarkSuite struct {
	Results []BenchmarkResult
	Started time.Time
}