package node_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"testing"
	"time"

	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/internal/node"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// Helper functions stay the same...
func createRandomTriple() Triple {
	var id [20]byte
	rand.Read(id[:])
	return Triple{
		ID:   id[:],
		Addr: Address{IP: "127.0.0.1", Port: 8000 + mathrand.Intn(1000)},
		Port: 8000 + mathrand.Intn(1000),
	}
}

func createTripleWithID(id []byte) Triple {
	fullID := make([]byte, 20)
	copy(fullID, id)
	return Triple{
		ID:   fullID,
		Addr: Address{IP: "127.0.0.1", Port: 8000},
		Port: 8000,
	}
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
	if node.Address().IP == "" {
		t.Error("Node address should be set")
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
	time.Sleep(50 * time.Millisecond)

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




func TestNodeStoreRetrieveIntegration(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	go node.Start()
	time.Sleep(50 * time.Millisecond)

	// Test multiple keys
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// Store all keys
	for key, value := range testData {
		node.StoreObject(key, value)
	}

	// Retrieve and verify all keys
	for key, expectedValue := range testData {
		retrievedValue, source, found := node.FindObject(key)
		if !found {
			t.Errorf("Key %s should be found", key)
		}
		if !bytes.Equal(retrievedValue, expectedValue) {
			t.Errorf("For key %s: expected %s, got %s", key, string(expectedValue), string(retrievedValue))
		}
		if source != node.Address().String() {
			t.Errorf("Source should be local node address")
		}
	}
}

func TestNodeHandlerErrors(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Register handler that returns an error
	node.Handle("error_test", func(msg Message) error {
		return fmt.Errorf("test error")
	})

	go node.Start()
	time.Sleep(50 * time.Millisecond)

	// Send message that will cause handler error (should not crash)
	err = node.Send(node.Address(), "error_test", []byte("data"))
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Node should still be responsive
	err = node.Send(node.Address(), "ping", []byte("ping"))
	if err != nil {
		t.Error("Node should still be responsive after handler error")
	}
}
