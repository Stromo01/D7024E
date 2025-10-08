package main

import (
	"testing"
)

// Configuration for easy testing parameter changes
var (
	TestNodeCount     = 100  // Easy to change for different test scales
	TestDropRate      = 0.05 // Easy to change packet drop percentage
	TestFailureRate   = 0.1  // Easy to change node failure rate
	QuickTestNodes    = 50   // For faster test iterations
	QuickTestDropRate = 0.02 // Lower drop rate for quick tests
)

func TestConfigurableScale(t *testing.T) {
	if testing.Short() {
		t.Logf("Running quick test with %d nodes, %.1f%% drop rate",
			QuickTestNodes, QuickTestDropRate*100)
		testLargeScaleNetwork(t, QuickTestNodes, QuickTestDropRate)
	} else {
		t.Logf("Running full scale test with %d nodes, %.1f%% drop rate",
			TestNodeCount, TestDropRate*100)
		testLargeScaleNetwork(t, TestNodeCount, TestDropRate)
	}
}

func TestConfigurableResilience(t *testing.T) {
	nodeCount := QuickTestNodes
	if !testing.Short() {
		nodeCount = TestNodeCount / 2 // Use half for resilience test
	}

	testNetworkResilience(t, nodeCount, TestDropRate, TestFailureRate)
}

// Benchmark for performance testing
func BenchmarkNetworkScale(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testLargeScaleNetwork(&testing.T{}, 50, 0.01) // Small scale for benchmarking
	}
}
