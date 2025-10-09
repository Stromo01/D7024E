package main

import (
    "testing"
)

// Configuration for easy testing parameter changes
var (
    // Fast test configuration
    FastTestNodes     = 1000
    FastTestDropRate  = 0.01

    // Thorough test configuration
    ConfigTestNodeCount = 1000
    ConfigTestDropRate  = 0.05
    ConfigTestFailureRate = 0.1

    // Quick dev configuration
    QuickTestNodes    = 100
    QuickTestDropRate = 0.02
)

func TestFastLargeScale(t *testing.T) {
    t.Logf("Running fast large scale test with %d nodes, %.1f%% drop rate",
        FastTestNodes, FastTestDropRate*100)
    testLargeScaleNetwork(t, FastTestNodes, FastTestDropRate)
}

func TestConfigurableScale(t *testing.T) {
    if testing.Short() {
        t.Logf("Running quick test with %d nodes, %.1f%% drop rate",
            QuickTestNodes, QuickTestDropRate*100)
        testLargeScaleNetwork(t, QuickTestNodes, QuickTestDropRate)
    } else {
        t.Logf("Running full scale test with %d nodes, %.1f%% drop rate",
            ConfigTestNodeCount, ConfigTestDropRate*100)
        testLargeScaleNetwork(t, ConfigTestNodeCount, ConfigTestDropRate)
    }
}

func TestConfigurableResilience(t *testing.T) {
    nodeCount := QuickTestNodes
    if !testing.Short() {
        nodeCount = ConfigTestNodeCount / 2 // Use half for resilience test
    }

    testNetworkResilience(t, nodeCount, ConfigTestDropRate, ConfigTestFailureRate)
}

// Benchmark for performance testing
func BenchmarkNetworkScale1000(b *testing.B) {
    for i := 0; i < b.N; i++ {
        benchmarkLargeScaleNetwork(b, 1000, 0.01)
    }
}

func benchmarkLargeScaleNetwork(b *testing.B, nodeCount int, dropRate float64) {
    network := NewEmulatedNetwork(dropRate)
    nodes := make([]*Node, nodeCount)

    // Create nodes
    for i := 0; i < nodeCount; i++ {
        addr := Address{IP: "127.0.0.1", Port: 30000 + i} // Different port range
        node, err := NewNode(network, addr)
        if err != nil {
            b.Fatalf("Failed to create node %d: %v", i, err)
        }
        nodes[i] = node
        node.Start()
    }

    // Minimal connection test - just connect first 10 nodes
    for i := 1; i < min(10, nodeCount); i++ {
        triple := Triple{
            ID:   nodes[0].id[:],
            Addr: nodes[0].addr,
            Port: nodes[0].addr.Port,
        }
        nodes[i].JoinNetwork(triple)
    }

    // Minimal wait
    //time.Sleep(100 * time.Millisecond) // Commented out for faster benchmarking

    // Cleanup
    for _, node := range nodes {
        node.Close()
    }
}