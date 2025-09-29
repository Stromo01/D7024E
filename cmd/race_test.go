package main

import (
	"fmt"
	"testing"
	"time"
)

func TestRPCChannelRace(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create two nodes
	nodeA, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9500})
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}

	nodeB, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9501})
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}

	// Start nodes
	nodeA.Start()
	nodeB.Start()

	// Store a value at node B
	nodeB.StoreObject("race-test-key", []byte("race-test-value"))

	time.Sleep(100 * time.Millisecond)

	// Test multiple sendRPCWithResponse calls in sequence
	for i := 0; i < 3; i++ {
		fmt.Printf("Attempt %d\n", i+1)

		req := FindValueRequest{Key: "race-test-key"}
		resp, err := nodeA.sendRPCWithResponse(nodeB.addr, MsgFindValue, req, 2*time.Second)

		if err != nil {
			t.Errorf("Attempt %d failed: %v", i+1, err)
		} else {
			fmt.Printf("Attempt %d succeeded with response: %v\n", i+1, resp)
		}

		// Small delay between attempts
		time.Sleep(10 * time.Millisecond)
	}

	// Clean up
	nodeA.Close()
	nodeB.Close()

	t.Log("RPC channel race test completed")
}
