package main

import (
	"fmt"
	"testing"
	"time"
)

func TestBasicRPC(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create two nodes
	nodeA, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9300})
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}

	nodeB, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9301})
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}

	// Start nodes
	nodeA.Start()
	nodeB.Start()

	// Test basic PING RPC (which we know works)
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("Testing PING from A to B\n")
	err = nodeA.RPCPing(nodeB.addr)
	if err != nil {
		t.Errorf("PING failed: %v", err)
	} else {
		fmt.Printf("PING succeeded\n")
	}

	// Test STORE RPC
	fmt.Printf("Testing STORE from A to B\n")
	err = nodeA.RPCStore(nodeB.addr, "test-key", []byte("test-value"))
	if err != nil {
		t.Errorf("STORE failed: %v", err)
	} else {
		fmt.Printf("STORE succeeded\n")
	}

	time.Sleep(200 * time.Millisecond)

	// Check if value was stored
	if value, found := nodeB.FindObject("test-key"); found {
		fmt.Printf("Value was stored: '%s'\n", string(value))
	} else {
		t.Error("Value was not stored")
	}

	// Clean up
	nodeA.Close()
	nodeB.Close()

	t.Log("Basic RPC test completed")
}
