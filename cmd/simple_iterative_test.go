package main

import (
	"testing"
	"time"
)

func TestIterativeSimple(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create two nodes
	nodeA, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9100})
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}

	nodeB, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9101})
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}

	// Start nodes
	nodeA.Start()
	nodeB.Start()

	// Add each other to routing tables
	contactB := Triple{
		ID:   nodeB.id[:],
		Addr: nodeB.addr,
		Port: nodeB.addr.Port,
	}
	nodeA.routing.addContact(contactB)

	contactA := Triple{
		ID:   nodeA.id[:],
		Addr: nodeA.addr,
		Port: nodeA.addr.Port,
	}
	nodeB.routing.addContact(contactA)

	// Store a value at node B
	testKey := "simple-test-key"
	testValue := []byte("simple-test-value")
	nodeB.StoreObject(testKey, testValue)

	// Wait for setup
	time.Sleep(100 * time.Millisecond)

	// Test iterative find value from node A
	foundValue, _, found := nodeA.IterativeFindValue(testKey)

	t.Logf("Found: %v, Value: '%s'", found, string(foundValue))

	if !found {
		t.Error("Should find the stored value")
	}

	if string(foundValue) != string(testValue) {
		t.Errorf("Expected value '%s', got '%s'", string(testValue), string(foundValue))
	}

	// Clean up
	nodeA.Close()
	nodeB.Close()

	t.Log("Simple iterative test completed")
}
