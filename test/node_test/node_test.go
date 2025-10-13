package node_test

import (
	"bytes"
	"testing"
	"time"

	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/internal/node"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// Helper function to create a random Triple for testing

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
}

