package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDirectRPCFindValue(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Create two nodes
	nodeA, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9200})
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}

	nodeB, err := NewNode(network, Address{IP: "127.0.0.1", Port: 9201})
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}

	// Start nodes
	nodeA.Start()
	nodeB.Start()

	// Store a value at node B
	testKey := "test-rpc-key"
	testValue := []byte("test-rpc-value")
	nodeB.StoreObject(testKey, testValue)

	// Wait for setup
	time.Sleep(100 * time.Millisecond)

	// Test direct RPC call from node A to node B
	req := FindValueRequest{Key: testKey}
	resp, err := nodeA.sendRPCWithResponse(nodeB.addr, MsgFindValue, req, 5*time.Second)

	t.Logf("RPC Response: %v, Error: %v", resp, err)

	if err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	// Parse the response
	if resp != nil {
		// Parse response properly using JSON unmarshaling
		data, marshalErr := json.Marshal(resp)
		if marshalErr == nil {
			var findValueResp FindValueResponse
			if unmarshalErr := json.Unmarshal(data, &findValueResp); unmarshalErr == nil {
				if findValueResp.Found {
					t.Logf("Found value: '%s'", string(findValueResp.Value))
					if string(findValueResp.Value) != string(testValue) {
						t.Errorf("Expected value '%s', got '%s'", string(testValue), string(findValueResp.Value))
					}
				} else {
					t.Error("Value should be found")
				}
			} else {
				t.Errorf("Failed to unmarshal response: %v", unmarshalErr)
			}
		} else {
			t.Errorf("Failed to marshal response: %v", marshalErr)
		}
	} else {
		t.Error("Response should not be nil")
	}

	// Clean up
	nodeA.Close()
	nodeB.Close()

	t.Log("Direct RPC test completed")
}
