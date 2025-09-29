package main

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

// LargeScaleNetwork simulates a network with configurable node count and packet dropping
type LargeScaleNetwork struct {
	baseNetwork    Network // Embedded network implementation
	packetDropRate float64 // 0.0 to 1.0 (0% to 100%)
	mu             sync.RWMutex
}

// NewLargeScaleNetwork creates a network simulation for large-scale testing
func NewLargeScaleNetwork(packetDropRate float64) *LargeScaleNetwork {
	return &LargeScaleNetwork{
		baseNetwork:    NewMockNetwork(),
		packetDropRate: packetDropRate,
	}
}

// SetPacketDropRate allows changing the packet drop rate during testing
func (n *LargeScaleNetwork) SetPacketDropRate(rate float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.packetDropRate = rate
}

// Listen delegates to the base network
func (n *LargeScaleNetwork) Listen(addr Address) (Connection, error) {
	return n.baseNetwork.Listen(addr)
}

// Partition delegates to the base network
func (n *LargeScaleNetwork) Partition(group1, group2 []Address) {
	n.baseNetwork.Partition(group1, group2)
}

// Heal delegates to the base network
func (n *LargeScaleNetwork) Heal() {
	n.baseNetwork.Heal()
}

// Dial wraps MockNetwork.Dial with packet dropping simulation
func (n *LargeScaleNetwork) Dial(addr Address) (Connection, error) {
	// Simulate packet drop during connection establishment
	n.mu.RLock()
	dropRate := n.packetDropRate
	n.mu.RUnlock()

	if rand.Float64() < dropRate {
		return nil, &NetworkError{Message: "Connection dropped (simulated packet loss)"}
	}

	conn, err := n.baseNetwork.Dial(addr)
	if err != nil {
		return nil, err
	}

	// Wrap the connection to simulate packet drops on sends
	return &PacketDroppingConnection{
		Connection: conn,
		dropRate:   dropRate,
		network:    n,
	}, nil
}

// PacketDroppingConnection wraps a connection to simulate packet loss
type PacketDroppingConnection struct {
	Connection
	dropRate float64
	network  *LargeScaleNetwork
}

// Send wraps Connection.Send with packet dropping simulation
func (c *PacketDroppingConnection) Send(msg Message) error {
	c.network.mu.RLock()
	dropRate := c.network.packetDropRate
	c.network.mu.RUnlock()

	if rand.Float64() < dropRate {
		// Simulate packet drop - don't send the message
		return &NetworkError{Message: "Packet dropped (simulated packet loss)"}
	}

	return c.Connection.Send(msg)
}

// NetworkError represents network-related errors
type NetworkError struct {
	Message string
}

func (e *NetworkError) Error() string {
	return e.Message
}

// TestLargeScaleNetwork tests the DHT with 1000+ nodes and packet dropping
func TestLargeScaleNetwork(t *testing.T) {
	// Test parameters - easily configurable
	nodeCount := 1000
	packetDropRate := 0.05 // 5% packet drop rate

	t.Logf("Starting large scale test with %d nodes and %.1f%% packet drop rate",
		nodeCount, packetDropRate*100)

	// Create large scale network with packet dropping
	network := NewLargeScaleNetwork(packetDropRate)

	// Create nodes
	nodes := make([]*Node, nodeCount)
	startTime := time.Now()

	t.Logf("Creating %d nodes...", nodeCount)
	for i := 0; i < nodeCount; i++ {
		addr := Address{
			IP:   "127.0.0.1",
			Port: 10000 + i, // Use ports 10000-10999+ for large scale test
		}

		node, err := NewNode(network, addr)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node

		// Start node
		node.Start()

		// Progress indicator
		if (i+1)%100 == 0 {
			t.Logf("Created %d/%d nodes", i+1, nodeCount)
		}
	}

	creationTime := time.Since(startTime)
	t.Logf("Created %d nodes in %v", nodeCount, creationTime)

	// Bootstrap network - connect nodes to form a network
	t.Logf("Bootstrapping network...")
	bootstrapStartTime := time.Now()

	// Use first 10 nodes as initial bootstrap nodes
	bootstrapNodes := nodes[:10]

	// Connect bootstrap nodes to each other
	for i, node := range bootstrapNodes {
		for j, otherNode := range bootstrapNodes {
			if i != j {
				contact := Triple{
					ID:   otherNode.id[:],
					Addr: otherNode.addr,
					Port: otherNode.addr.Port,
				}
				node.routing.addContact(contact)
			}
		}
	}

	// Connect remaining nodes to random bootstrap nodes
	for i := 10; i < nodeCount; i++ {
		// Connect to 3 random bootstrap nodes
		for j := 0; j < 3 && j < len(bootstrapNodes); j++ {
			bootstrapNode := bootstrapNodes[rand.Intn(len(bootstrapNodes))]
			contact := Triple{
				ID:   bootstrapNode.id[:],
				Addr: bootstrapNode.addr,
				Port: bootstrapNode.addr.Port,
			}
			nodes[i].routing.addContact(contact)
		}

		if (i+1)%100 == 0 {
			t.Logf("Bootstrapped %d/%d nodes", i+1, nodeCount)
		}
	}

	bootstrapTime := time.Since(bootstrapStartTime)
	t.Logf("Bootstrapped network in %v", bootstrapTime)

	// Wait for network to stabilize
	time.Sleep(500 * time.Millisecond)

	// Test basic operations with packet dropping
	t.Logf("Testing operations with packet dropping...")

	// Test 1: Store operations
	testKey := "large-scale-test-key"
	testValue := []byte("large-scale-test-value")

	storeNode := nodes[rand.Intn(nodeCount)]
	err := storeNode.IterativeStore(testKey, testValue)
	if err != nil {
		t.Logf("Store operation failed (expected with packet drops): %v", err)
	} else {
		t.Logf("Store operation succeeded despite packet drops")
	}

	// Test 2: Find operations
	findNode := nodes[rand.Intn(nodeCount)]
	value, _, found := findNode.IterativeFindValue(testKey)
	if found {
		t.Logf("Find operation succeeded: found value '%s'", string(value))
		if string(value) != string(testValue) {
			t.Errorf("Retrieved value doesn't match stored value")
		}
	} else {
		t.Logf("Find operation failed (expected with packet drops)")
	}

	// Test 3: Network resilience with higher packet drop rate
	t.Logf("Testing resilience with higher packet drop rate...")
	network.SetPacketDropRate(0.20) // Increase to 20% packet drop

	// Try multiple operations to test resilience
	successCount := 0
	totalTests := 10

	for i := 0; i < totalTests; i++ {
		testNode := nodes[rand.Intn(nodeCount)]
		targetKey := "resilience-test-key"
		targetValue := []byte("resilience-test-value")

		err := testNode.IterativeStore(targetKey, targetValue)
		if err == nil {
			successCount++
		}
	}

	successRate := float64(successCount) / float64(totalTests)
	t.Logf("Success rate with 20%% packet drop: %.1f%% (%d/%d operations succeeded)",
		successRate*100, successCount, totalTests)

	// Clean up
	t.Logf("Cleaning up %d nodes...", nodeCount)
	cleanupStart := time.Now()

	for i, node := range nodes {
		node.Close()
		if (i+1)%100 == 0 {
			t.Logf("Cleaned up %d/%d nodes", i+1, nodeCount)
		}
	}

	cleanupTime := time.Since(cleanupStart)
	t.Logf("Cleanup completed in %v", cleanupTime)

	totalTime := time.Since(startTime)
	t.Logf("Large scale test completed in %v", totalTime)
}

// TestConfigurableNetwork demonstrates easy configuration of node count and packet drop rate
func TestConfigurableNetwork(t *testing.T) {
	testCases := []struct {
		name           string
		nodeCount      int
		packetDropRate float64
	}{
		{"Small network, no drops", 50, 0.0},
		{"Medium network, light drops", 100, 0.02},
		{"Large network, moderate drops", 500, 0.05},
		{"Extreme network, heavy drops", 1000, 0.10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing %d nodes with %.1f%% packet drop rate",
				tc.nodeCount, tc.packetDropRate*100)

			network := NewLargeScaleNetwork(tc.packetDropRate)

			// Create and test a subset of operations for faster testing
			nodes := make([]*Node, tc.nodeCount)

			// Create nodes
			for i := 0; i < tc.nodeCount; i++ {
				addr := Address{
					IP:   "127.0.0.1",
					Port: 20000 + i, // Use different port range
				}

				node, err := NewNode(network, addr)
				if err != nil {
					t.Fatalf("Failed to create node %d: %v", i, err)
				}
				nodes[i] = node
				node.Start()
			}

			// Simple bootstrap - connect first few nodes
			if tc.nodeCount > 1 {
				for i := 0; i < min(5, tc.nodeCount-1); i++ {
					contact := Triple{
						ID:   nodes[i+1].id[:],
						Addr: nodes[i+1].addr,
						Port: nodes[i+1].addr.Port,
					}
					nodes[i].routing.addContact(contact)
				}
			}

			// Test basic functionality
			if tc.nodeCount >= 2 {
				err := nodes[0].IterativeStore("test-key", []byte("test-value"))
				if err != nil {
					t.Logf("Store failed (expected with packet drops): %v", err)
				}
			}

			// Clean up
			for _, node := range nodes {
				node.Close()
			}

			t.Logf("Test case '%s' completed successfully", tc.name)
		})
	}
}

// Helper function for minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BenchmarkLargeScaleOperations benchmarks DHT operations at scale
func BenchmarkLargeScaleOperations(b *testing.B) {
	nodeCount := 100                     // Smaller for benchmarking
	network := NewLargeScaleNetwork(0.0) // No packet drops for consistent benchmarking

	// Setup nodes
	nodes := make([]*Node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		addr := Address{
			IP:   "127.0.0.1",
			Port: 30000 + i,
		}

		node, err := NewNode(network, addr)
		if err != nil {
			b.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		node.Start()
	}

	// Bootstrap
	for i := 0; i < min(10, nodeCount-1); i++ {
		contact := Triple{
			ID:   nodes[i+1].id[:],
			Addr: nodes[i+1].addr,
			Port: nodes[i+1].addr.Port,
		}
		nodes[i].routing.addContact(contact)
	}

	b.ResetTimer()

	// Benchmark store operations
	b.Run("Store", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			node := nodes[i%nodeCount]
			key := "benchmark-key-" + string(rune(i))
			value := []byte("benchmark-value")

			_ = node.IterativeStore(key, value)
		}
	})

	// Benchmark find operations
	b.Run("Find", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			node := nodes[i%nodeCount]
			key := "benchmark-key-" + string(rune(i%100)) // Find existing keys

			_, _, _ = node.IterativeFindValue(key)
		}
	})

	// Cleanup
	for _, node := range nodes {
		node.Close()
	}
}
