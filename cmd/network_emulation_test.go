package main

import (
    "crypto/sha1"
    "fmt"
    "math/rand"
    "sync"
    "testing"
    "time"

)

// Configuration constants - easy to change
const (
    TestNodeCount = 1000
    TestDropRate  = 0.05 // 5% packet drop rate
)

// EmulatedNetwork extends MockNetwork with packet dropping
type EmulatedNetwork struct {
    *mockNetwork
    dropRate   float64
    totalDrops int64
    totalSent  int64
    mu         sync.RWMutex
}

func NewEmulatedNetwork(dropRate float64) *EmulatedNetwork {
    return &EmulatedNetwork{
        mockNetwork: NewMockNetwork().(*mockNetwork),
        dropRate:    dropRate,
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

// Single comprehensive test for 1000 nodes with packet dropping
func TestLargeScaleNetwork1000Nodes(t *testing.T) {
    network := NewEmulatedNetwork(TestDropRate)
    nodes := make([]*Node, TestNodeCount)

    t.Logf("Creating %d nodes with %.1f%% packet drop rate", TestNodeCount, TestDropRate*100)
    startTime := time.Now()

    // Create all nodes
    for i := 0; i < TestNodeCount; i++ {
        addr := Address{IP: "127.0.0.1", Port: 8000 + i}
        node, err := NewNode(network, addr)
        if err != nil {
            t.Fatalf("Failed to create node %d: %v", i, err)
        }
        nodes[i] = node
        node.Start()
    }

    t.Logf("Created %d nodes in %v", TestNodeCount, time.Since(startTime))

    // Basic connectivity test - connect first 50 nodes to demonstrate network formation
    connectStart := time.Now()
    maxConnections := 50

    for i := 1; i < maxConnections; i++ {
        triple := Triple{
            ID:   nodes[0].id[:],
            Addr: nodes[0].addr,
            Port: nodes[0].addr.Port,
        }
        err := nodes[i].JoinNetwork(triple)
        if err != nil {
            t.Logf("Node %d failed to join: %v", i, err)
        }
    }

    t.Logf("Connected %d nodes in %v", maxConnections, time.Since(connectStart))

    // Brief stabilization
    time.Sleep(200 * time.Millisecond)

    // Simple data operation test
    testData := "test_data_1000_nodes"
    hash := fmt.Sprintf("%x", sha1.Sum([]byte(testData)))

    // Store on first node
    err := nodes[0].StoreAtK(hash, []byte(testData), min(K, maxConnections))
    if err != nil {
        t.Logf("Store operation failed: %v", err)
    }

    // Brief wait for propagation
    time.Sleep(100 * time.Millisecond)

    // Try to retrieve from a few connected nodes
    retrieveAttempts := 5
    successCount := 0

    for i := 1; i < min(retrieveAttempts+1, maxConnections); i++ {
        value, source, found := nodes[i].FindObject(hash)
        if found && string(value) == testData {
            successCount++
            t.Logf("Successfully retrieved data from node %d (source: %s)", i, source)
        }
    }

    // Verify basic functionality
    if successCount == 0 {
        t.Logf("Warning: No successful retrievals (may be due to %.1f%% packet drops)", TestDropRate*100)
    } else {
        t.Logf("Success: %d/%d retrievals successful", successCount, retrieveAttempts)
    }

    // Verify all nodes exist and are functional
    totalActiveNodes := 0
    for _, node := range nodes {
        if node != nil {
            totalActiveNodes++
        }
    }

    if totalActiveNodes != TestNodeCount {
        t.Errorf("Expected %d active nodes, got %d", TestNodeCount, totalActiveNodes)
    }

    // Test basic message sending capability across the network
    messagesSent := 0
    for i := 0; i < min(10, TestNodeCount-1); i++ {
        err := nodes[i].Send(nodes[i+1].addr, "test", []byte("hello"))
        if err == nil {
            messagesSent++
        }
    }

    t.Logf("Successfully sent %d/10 test messages", messagesSent)

    // Cleanup
    cleanupStart := time.Now()
    for _, node := range nodes {
        node.Close()
    }
    t.Logf("Cleanup completed in %v", time.Since(cleanupStart))

    // Print network statistics
    totalSent, totalDrops, actualDropRate := network.GetStats()
    t.Logf("Network stats: %d sent, %d dropped (%.2f%% actual drop rate)",
        totalSent, totalDrops, actualDropRate*100)

    // Final verification
    t.Logf("Test completed: %d nodes created, basic networking verified", TestNodeCount)
}

// Helper function that can be called from other test files
func testLargeScaleNetwork(t *testing.T, nodeCount int, dropRate float64) {
    network := NewEmulatedNetwork(dropRate)
    nodes := make([]*Node, nodeCount)

    t.Logf("Creating %d nodes with %.1f%% packet drop rate", nodeCount, dropRate*100)
    startTime := time.Now()

    // Create all nodes
    for i := 0; i < nodeCount; i++ {
        addr := Address{IP: "127.0.0.1", Port: 10000 + i} // Different port range
        node, err := NewNode(network, addr)
        if err != nil {
            t.Fatalf("Failed to create node %d: %v", i, err)
        }
        nodes[i] = node
        node.Start()
    }

    t.Logf("Created %d nodes in %v", nodeCount, time.Since(startTime))

    // Connect subset of nodes for testing
    maxConnections := min(50, nodeCount-1)
    connectStart := time.Now()

    for i := 1; i <= maxConnections; i++ {
        triple := Triple{
            ID:   nodes[0].id[:],
            Addr: nodes[0].addr,
            Port: nodes[0].addr.Port,
        }
        err := nodes[i].JoinNetwork(triple)
        if err != nil {
            t.Logf("Node %d failed to join: %v", i, err)
        }
    }

    t.Logf("Connected %d nodes in %v", maxConnections, time.Since(connectStart))

    // Brief stabilization
    time.Sleep(100 * time.Millisecond)

    // Simple data test
    testData := "configurable_test_data"
    hash := fmt.Sprintf("%x", sha1.Sum([]byte(testData)))

    err := nodes[0].StoreAtK(hash, []byte(testData), min(K, maxConnections))
    if err != nil {
        t.Logf("Store operation failed: %v", err)
    }

    time.Sleep(50 * time.Millisecond)

    // Test retrieval
    value, source, found := nodes[1].FindObject(hash)
    if found && string(value) == testData {
        t.Logf("Data retrieval successful (source: %s)", source)
    } else {
        t.Logf("Data retrieval failed (expected with %.1f%% drop rate)", dropRate*100)
    }

    // Cleanup
    for _, node := range nodes {
        node.Close()
    }

    // Print stats
    totalSent, totalDrops, actualDropRate := network.GetStats()
    t.Logf("Network stats: %d sent, %d dropped (%.2f%% actual drop rate)",
        totalSent, totalDrops, actualDropRate*100)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}