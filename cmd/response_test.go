package main

import (
	"fmt"
	"testing"
	"time"
)

func TestRPCResponseMatching(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create two nodes
	nodeA, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9400})
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}

	nodeB, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9401})
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}

	// Start nodes
	nodeA.Start()
	nodeB.Start()

	// Store a value at node B first
	nodeB.StoreObject("test-response-key", []byte("test-response-value"))

	time.Sleep(100 * time.Millisecond)

	// Create and send a FIND_VALUE RPC manually to debug
	req := FindValueRequest{Key: "test-response-key"}

	fmt.Printf("Sending FIND_VALUE RPC...\n")
	rpcID, err := SendRPC(nodeA.network, nodeA.addr, nodeB.addr, MsgFindValue, req)
	if err != nil {
		t.Fatalf("Failed to send RPC: %v", err)
	}

	fmt.Printf("Sent RPC with ID: %v\n", rpcID)

	// Wait a bit for response
	time.Sleep(200 * time.Millisecond)

	// Check if we got a response in our pending RPCs
	nodeA.rpcMu.RLock()
	_, exists := nodeA.pendingRPCs[rpcID]
	nodeA.rpcMu.RUnlock()

	if exists {
		fmt.Printf("Response channel still exists (not matched)\n")
	} else {
		fmt.Printf("Response channel was removed (response was matched)\n")
	}

	// Now test with sendRPCWithResponse
	fmt.Printf("Testing sendRPCWithResponse...\n")
	resp, err := nodeA.sendRPCWithResponse(nodeB.addr, MsgFindValue, req, 1*time.Second)

	if err != nil {
		t.Errorf("sendRPCWithResponse failed: %v", err)
	} else {
		fmt.Printf("Response received: %v\n", resp)

		// Try to parse as FindValueResponse
		if respMap, ok := resp.(map[string]interface{}); ok {
			if found, exists := respMap["found"].(bool); exists {
				fmt.Printf("Found field: %v\n", found)
				if found {
					if value, hasValue := respMap["value"]; hasValue {
						fmt.Printf("Value: %v\n", value)
					}
				}
			}
		}
	}

	// Clean up
	nodeA.Close()
	nodeB.Close()

	t.Log("RPC response matching test completed")
}
