package main

import (
	"crypto/sha1"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

const (
	DefaultNodeCount   = 1000
	DefaultDropRate    = 0.05 // 5% packet drop rate
	DefaultTestTimeout = 30 * time.Second
)

// EmulatedNetwork extends MockNetwork with packet dropping and large scale support
type EmulatedNetwork struct {
	*mockNetwork
	dropRate   float64
	nodeCount  int
	totalDrops int64
	totalSent  int64
	mu         sync.RWMutex
}

func NewEmulatedNetwork(nodeCount int, dropRate float64) *EmulatedNetwork {
	return &EmulatedNetwork{
		mockNetwork: NewMockNetwork().(*mockNetwork),
		dropRate:    dropRate,
		nodeCount:   nodeCount,
	}
}

func (n *EmulatedNetwork) shouldDropPacket() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.totalSent++

	if rand.Float64() < n.dropRate {
		n.totalDrops++
		return true
	}
	return false
}

func (n *EmulatedNetwork) GetStats() (totalSent, totalDrops int64, actualDropRate float64) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.totalSent == 0 {
		return 0, 0, 0
	}

	return n.totalSent, n.totalDrops, float64(n.totalDrops) / float64(n.totalSent)
}

// Override Dial to add packet dropping
func (n *EmulatedNetwork) Dial(addr Address) (Connection, error) {
	conn, err := n.mockNetwork.Dial(addr)
	if err != nil {
		return nil, err
	}

	return &emulatedConnection{
		mockConnection: conn.(*mockConnection),
		network:        n,
	}, nil
}

type emulatedConnection struct {
	*mockConnection
	network *EmulatedNetwork
}

func (c *emulatedConnection) Send(msg Message) error {
	// Check if packet should be dropped
	if c.network.shouldDropPacket() {
		// Simulate packet drop by not sending
		return nil // Return nil to simulate successful send from sender's perspective
	}

	return c.mockConnection.Send(msg)
}

func TestLargeScaleNetworkEmulation(t *testing.T) {
	testCases := []struct {
		name      string
		nodeCount int
		dropRate  float64
	}{
		{"1000_nodes_no_drops", 1000, 0.0},
		{"1000_nodes_5pct_drops", 1000, 0.05},
		{"500_nodes_10pct_drops", 500, 0.10},
		{"100_nodes_20pct_drops", 100, 0.20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testLargeScaleNetwork(t, tc.nodeCount, tc.dropRate)
		})
	}
}

func testLargeScaleNetwork(t *testing.T, nodeCount int, dropRate float64) {
	network := NewEmulatedNetwork(nodeCount, dropRate)
	nodes := make([]*Node, nodeCount)

	// Create nodes
	t.Logf("Creating %d nodes with %.1f%% packet drop rate", nodeCount, dropRate*100)
	startTime := time.Now()

	for i := 0; i < nodeCount; i++ {
		addr := Address{IP: "127.0.0.1", Port: 8000 + i}
		node, err := NewNode(network, addr)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		node.Start()
	}

	t.Logf("Created %d nodes in %v", nodeCount, time.Since(startTime))

	// Connect nodes to bootstrap (first 10 nodes act as initial bootstrap ring)
	connectStart := time.Now()
	bootstrapNodes := min(10, nodeCount)

	// Connect first few nodes to each other to form initial ring
	for i := 1; i < bootstrapNodes; i++ {
		triple := Triple{
			ID:   nodes[0].id[:],
			Addr: nodes[0].addr,
			Port: nodes[0].addr.Port,
		}
		err := nodes[i].JoinNetwork(triple)
		if err != nil {
			t.Logf("Node %d failed to join via bootstrap: %v", i, err)
		}
	}

	// Connect remaining nodes to random bootstrap nodes
	for i := bootstrapNodes; i < nodeCount; i++ {
		bootstrapIdx := rand.Intn(bootstrapNodes)
		triple := Triple{
			ID:   nodes[bootstrapIdx].id[:],
			Addr: nodes[bootstrapIdx].addr,
			Port: nodes[bootstrapIdx].addr.Port,
		}
		err := nodes[i].JoinNetwork(triple)
		if err != nil {
			t.Logf("Node %d failed to join via node %d: %v", i, bootstrapIdx, err)
		}
	}

	t.Logf("Connected nodes in %v", time.Since(connectStart))

	// Wait for network to stabilize
	time.Sleep(2 * time.Second)

	// Test data storage and retrieval
	testDataOperations(t, nodes[:min(50, nodeCount)], network, dropRate)

	// Cleanup
	for _, node := range nodes {
		node.Close()
	}

	// Print network statistics
	totalSent, totalDrops, actualDropRate := network.GetStats()
	t.Logf("Network stats: %d sent, %d dropped (%.2f%% actual drop rate)",
		totalSent, totalDrops, actualDropRate*100)
}

func testDataOperations(t *testing.T, nodes []*Node, network *EmulatedNetwork, expectedDropRate float64) {
	if len(nodes) == 0 {
		return
	}

	// Store test data
	testData := "large_scale_test_data"
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(testData)))

	// Store from random node
	storeNode := nodes[rand.Intn(len(nodes))]
	err := storeNode.StoreAtK(hash, []byte(testData), K)
	if err != nil {
		t.Logf("Failed to store data: %v", err)
	}

	// Wait for storage to propagate
	time.Sleep(1 * time.Second)

	// Try to retrieve from multiple nodes
	successCount := 0
	for i := 0; i < min(10, len(nodes)); i++ {
		retrieveNode := nodes[rand.Intn(len(nodes))]
		value, source, found := retrieveNode.FindObject(hash)

		if found && string(value) == testData {
			successCount++
			t.Logf("Successfully retrieved data from node %s (source: %s)",
				retrieveNode.Address().String(), source)
		}
	}

	// With packet drops, we expect some failures, but not total failure
	minExpectedSuccess := int(float64(10) * (1.0 - expectedDropRate*2)) // Allow for 2x drop rate impact
	if successCount < minExpectedSuccess {
		t.Logf("Warning: Only %d/10 retrievals successful (expected >= %d with %.1f%% drop rate)",
			successCount, minExpectedSuccess, expectedDropRate*100)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
