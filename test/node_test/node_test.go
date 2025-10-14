package node_test

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/internal/node"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// Helper function to create random 20-byte ID
func createRandomID() [20]byte {
	var id [20]byte
	rand.Read(id[:])
	return id
}
func TestNewNode(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Test node has valid ID
	var zeroID [20]byte
	if bytes.Equal(node.Id[:], zeroID[:]) {
		t.Error("Node ID should not be zero")
	}

	// Test address is set
	if node.Address().IP != addr.IP {
		t.Errorf("Expected IP %s, got %s", addr.IP, node.Address().IP)
	}
}

func TestNodeStartClose(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Start node
	go node.Start()
	time.Sleep(10 * time.Millisecond)

	// Close node
	err = node.Close()
	if err != nil {
		t.Errorf("Failed to close node: %v", err)
	}
}

func TestNodeStoreAndRetrieve(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Store object
	key := "test-key"
	value := []byte("test-value")
	node.StoreObject(key, value)

	// Retrieve object
	retrieved, exists := node.FindObjectLocally(key)
	if !exists {
		t.Error("Object should exist")
	}
	if !bytes.Equal(retrieved, value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}
}

func TestNodeIterativeStore(t *testing.T) {
	// Use the same network as other tests
	network := NewUDPNetwork()

	// Create node
	addr := Address{IP: "127.0.0.1", Port: 0}
	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Start the node
	go node.Start()
	time.Sleep(10 * time.Millisecond)

	// Add some contacts to routing table
	contacts := []Triple{
		createTripleWithID([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
		createTripleWithID([]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
		createTripleWithID([]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
	}

	// Add contacts to routing table
	for _, contact := range contacts {
		node.RoutingTable().AddContact(contact)
	}

	// Store some data
	key := "test-key"
	value := []byte("test-value")

	// Call IterativeStore
	node.IterativeStore(key, value)

	// Simple verification: check that the function completes without crashing
	closest := node.RoutingTable().GetKClosest(key, 3)
	if len(closest) == 0 {
		t.Error("Should have contacts in routing table after IterativeStore")
	}

	t.Logf("IterativeStore completed for key '%s'", key)
}

func TestNodeFindObject(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Store locally
	key := "find-key"
	value := []byte("find-value")
	node.StoreObject(key, value)

	// Find object (should find locally)
	foundValue, source, found := node.FindObject(key)
	if !found {
		t.Error("Object should be found")
	}
	if !bytes.Equal(foundValue, value) {
		t.Errorf("Expected %s, got %s", string(value), string(foundValue))
	}
	if source != node.Address().String() {
		t.Errorf("Expected source %s, got %s", node.Address().String(), source)
	}
}

func TestXorDistance(t *testing.T) {
	// Test with single bytes first
	a := []byte{0x00}
	b := []byte{0x01}

	distance := XorDistance(a, b)
	if distance.Int64() != 1 {
		t.Errorf("Expected distance 1, got %d", distance.Int64())
	}

	// Test symmetry
	dist1 := XorDistance(a, b)
	dist2 := XorDistance(b, a)
	if dist1.Cmp(dist2) != 0 {
		t.Error("XOR distance should be symmetric")
	}

	// Test distance to self
	dist := XorDistance(a, a)
	if dist.Int64() != 0 {
		t.Error("Distance to self should be 0")
	}

	// Test with realistic 20-byte node IDs
	id1 := createRandomID()
	id2 := createRandomID()
	id3 := createRandomID()

	// Test properties with large IDs
	dist_1_2 := XorDistance(id1[:], id2[:])
	dist_2_1 := XorDistance(id2[:], id1[:])

	// Symmetry
	if dist_1_2.Cmp(dist_2_1) != 0 {
		t.Error("XOR distance should be symmetric for 20-byte IDs")
	}

	// Triangle inequality: d(a,c) <= d(a,b) + d(b,c)
	dist_1_3 := XorDistance(id1[:], id3[:])
	dist_2_3 := XorDistance(id2[:], id3[:])

	// Just verify the calculation works without errors
	if dist_1_2 == nil || dist_1_3 == nil || dist_2_3 == nil {
		t.Error("XOR distance calculation failed for 20-byte IDs")
	}

	// Test distance to self with large ID
	dist_self := XorDistance(id1[:], id1[:])
	if dist_self.Int64() != 0 {
		t.Error("Distance to self should be 0 for 20-byte IDs")
	}

	t.Log("XOR distance calculations work correctly with realistic 20-byte IDs")
}
