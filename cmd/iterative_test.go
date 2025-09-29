package main

import (
	"fmt"
	"testing"
	"time"
)

func TestIterativeFindNode(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create test nodes
	nodes := make([]*Node, 5)
	addresses := []Address{
		{IP: "127.0.0.1", Port: 9000},
		{IP: "127.0.0.1", Port: 9001},
		{IP: "127.0.0.1", Port: 9002},
		{IP: "127.0.0.1", Port: 9003},
		{IP: "127.0.0.1", Port: 9004},
	}

	// Initialize nodes
	for i := 0; i < 5; i++ {
		node, err := NewNode(network, addresses[i])
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		// Start the node to handle messages
		node.Start()
	}

	// Let nodes know about each other (simulate bootstrap)
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i != j {
				contact := Triple{
					ID:   nodes[j].id[:],
					Addr: nodes[j].addr,
					Port: nodes[j].addr.Port,
				}
				nodes[i].routing.addContact(contact)
			}
		}
	}

	// Wait a bit for setup
	time.Sleep(100 * time.Millisecond)

	// Test iterative find node - look for node 2's ID from node 0
	targetID := nodes[2].id[:]
	foundContacts := nodes[0].IterativeFindNode(targetID)

	// Should find some contacts
	if len(foundContacts) == 0 {
		t.Error("IterativeFindNode should return at least one contact")
	}

	// Should find the target node
	foundTarget := false
	for _, contact := range foundContacts {
		if string(contact.ID) == string(targetID) {
			foundTarget = true
			break
		}
	}

	if !foundTarget {
		t.Error("IterativeFindNode should find the target node")
	}

	// Clean up
	for i := 0; i < 5; i++ {
		nodes[i].Close()
	}

	t.Log("IterativeFindNode test completed successfully")
}

func TestIterativeFindValue(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create test nodes
	nodes := make([]*Node, 3)
	addresses := []Address{
		{IP: "127.0.0.1", Port: 9010},
		{IP: "127.0.0.1", Port: 9011},
		{IP: "127.0.0.1", Port: 9012},
	}

	// Initialize nodes
	for i := 0; i < 3; i++ {
		node, err := NewNode(network, addresses[i])
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		// Start the node to handle messages
		node.Start()
	}

	// Let nodes know about each other
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i != j {
				contact := Triple{
					ID:   nodes[j].id[:],
					Addr: nodes[j].addr,
					Port: nodes[j].addr.Port,
				}
				nodes[i].routing.addContact(contact)
			}
		}
	}

	// Store a value at node 1
	testKey := "test-iterative-key"
	testValue := []byte("test-iterative-value")
	nodes[1].StoreObject(testKey, testValue)

	// Wait a bit for setup
	time.Sleep(100 * time.Millisecond)

	// Test 1: Find existing value from node 0
	foundValue, _, found := nodes[0].IterativeFindValue(testKey)

	if !found {
		t.Error("IterativeFindValue should find the stored value")
	}

	if string(foundValue) != string(testValue) {
		t.Errorf("Expected value '%s', got '%s'", string(testValue), string(foundValue))
	}

	// Test 2: Find non-existing value
	nonExistentKey := "non-existent-key"
	_, contacts2, found2 := nodes[0].IterativeFindValue(nonExistentKey)

	if found2 {
		t.Error("IterativeFindValue should not find non-existent value")
	}

	if len(contacts2) == 0 {
		t.Error("IterativeFindValue should return contacts when value not found")
	}

	// Clean up
	for i := 0; i < 3; i++ {
		nodes[i].Close()
	}

	t.Log("IterativeFindValue test completed successfully")
}

func TestIterativeStore(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create test nodes
	nodes := make([]*Node, 4)
	addresses := []Address{
		{IP: "127.0.0.1", Port: 9020},
		{IP: "127.0.0.1", Port: 9021},
		{IP: "127.0.0.1", Port: 9022},
		{IP: "127.0.0.1", Port: 9023},
	}

	// Initialize nodes
	for i := 0; i < 4; i++ {
		node, err := NewNode(network, addresses[i])
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		// Start the node to handle messages
		node.Start()
	}

	// Let nodes know about each other
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if i != j {
				contact := Triple{
					ID:   nodes[j].id[:],
					Addr: nodes[j].addr,
					Port: nodes[j].addr.Port,
				}
				nodes[i].routing.addContact(contact)
			}
		}
	}

	// Wait a bit for setup
	time.Sleep(100 * time.Millisecond)

	// Test iterative store
	testKey := "iterative-store-key"
	testValue := []byte("iterative-store-value")

	err := nodes[0].IterativeStore(testKey, testValue)
	if err != nil {
		t.Errorf("IterativeStore failed: %v", err)
	}

	// Wait for storage to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that the value was stored on multiple nodes
	storeCount := 0
	for i := 1; i < 4; i++ { // Skip node 0 (the originator)
		if value, found := nodes[i].FindObject(testKey); found {
			if string(value) == string(testValue) {
				storeCount++
			}
		}
	}

	if storeCount == 0 {
		t.Error("IterativeStore should store value on at least one remote node")
	}

	fmt.Printf("Value stored on %d nodes\n", storeCount)

	// Clean up
	for i := 0; i < 4; i++ {
		nodes[i].Close()
	}

	t.Log("IterativeStore test completed successfully")
}

func TestIterativeOperationsIntegration(t *testing.T) {
	// Test the full cycle: iterative store -> iterative find value

	// Create a mock network
	network := NewMockNetwork()

	// Create test nodes
	nodes := make([]*Node, 5)
	addresses := []Address{
		{IP: "127.0.0.1", Port: 9030},
		{IP: "127.0.0.1", Port: 9031},
		{IP: "127.0.0.1", Port: 9032},
		{IP: "127.0.0.1", Port: 9033},
		{IP: "127.0.0.1", Port: 9034},
	}

	// Initialize nodes
	for i := 0; i < 5; i++ {
		node, err := NewNode(network, addresses[i])
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		// Start the node to handle messages
		node.Start()
	}

	// Let nodes know about each other
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i != j {
				contact := Triple{
					ID:   nodes[j].id[:],
					Addr: nodes[j].addr,
					Port: nodes[j].addr.Port,
				}
				nodes[i].routing.addContact(contact)
			}
		}
	}

	// Wait for setup
	time.Sleep(100 * time.Millisecond)

	// Store value using iterative store from node 0
	testKey := "integration-test-key"
	testValue := []byte("integration-test-value")

	err := nodes[0].IterativeStore(testKey, testValue)
	if err != nil {
		t.Fatalf("IterativeStore failed: %v", err)
	}

	// Wait for storage to propagate
	time.Sleep(300 * time.Millisecond)

	// Find value using iterative find from node 4 (different node)
	foundValue, _, found := nodes[4].IterativeFindValue(testKey)

	if !found {
		t.Error("IterativeFindValue should find the value stored by IterativeStore")
	}

	if string(foundValue) != string(testValue) {
		t.Errorf("Expected value '%s', got '%s'", string(testValue), string(foundValue))
	}

	// Clean up
	for i := 0; i < 5; i++ {
		nodes[i].Close()
	}

	t.Log("Iterative operations integration test completed successfully")
}
