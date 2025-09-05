package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Removed duplicate Address.String method to avoid conflict with the existing declaration.

func TestHelloworld(t *testing.T) {
	// Create network and nodes
	network := NewMockNetwork()
	alice, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8080})
	bob, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8081})

	// Channel for synchronization
	done := make(chan struct{})

	// Alice says hello when she receives a message
	alice.Handle("hello", func(msg Message) error {
		fmt.Printf("Alice: Hello %s!\n", msg.From.IP)
		return msg.ReplyString("reply", "Nice to meet you!")
	})

	// Bob prints replies
	bob.Handle("reply", func(msg Message) error {
		fmt.Printf("Bob: %s\n", string(msg.Payload)[6:]) // Skip "reply:" prefix
		done <- struct{}{}
		return nil
	})

	// Start nodes and send message
	alice.Start()
	bob.Start()
	bob.SendString(alice.Address(), "hello", "Hi Alice!")

	// Wait for completion and cleanup
	<-done
	alice.Close()
	bob.Close()
	fmt.Println("Done!")
}

func TestNodeCommunication(t *testing.T) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a mock network
	network := NewMockNetwork()

	// Define addresses
	nodeAAddr := Address{IP: "127.0.0.1", Port: 8080}
	nodeBAddr := Address{IP: "127.0.0.1", Port: 8081}

	// Create nodes
	nodeA, err := NewNode(network, nodeAAddr)
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}

	nodeB, err := NewNode(network, nodeBAddr)
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}

	// Channels for synchronization
	nodeAReceived := make(chan struct{}, 2)
	nodeBReceived := make(chan struct{}, 2)

	// Helper function to extract message content after ":"
	extractContent := func(payload []byte) string {
		payloadStr := string(payload)
		for i, char := range payloadStr {
			if char == ':' {
				return payloadStr[i+1:]
			}
		}
		return payloadStr
	}

	// Helper function to wait for channel with timeout
	waitForSignal := func(ch <-chan struct{}, description string) {
		select {
		case <-ch:
			t.Logf("✓ %s", description)
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for: %s", description)
		}
	}

	// Register message handlers for node A
	nodeA.Handle("ping", func(msg Message) error {
		content := extractContent(msg.Payload)
		t.Logf("Node A received ping from %s: %s", msg.From.String(), content)
		nodeAReceived <- struct{}{}

		// Send pong back
		return nodeA.SendString(msg.From, "pong", "pong response from A")
	})

	nodeA.Handle("pong", func(msg Message) error {
		content := extractContent(msg.Payload)
		t.Logf("Node A received pong from %s: %s", msg.From.String(), content)
		nodeAReceived <- struct{}{}
		return nil
	})

	// Register message handlers for node B
	nodeB.Handle("ping", func(msg Message) error {
		content := extractContent(msg.Payload)
		t.Logf("Node B received ping from %s: %s", msg.From.String(), content)
		nodeBReceived <- struct{}{}

		// Send pong back
		return nodeB.SendString(msg.From, "pong", "pong response from B")
	})

	nodeB.Handle("pong", func(msg Message) error {
		content := extractContent(msg.Payload)
		t.Logf("Node B received pong from %s: %s", msg.From.String(), content)
		nodeBReceived <- struct{}{}
		return nil
	})

	// Start both nodes
	nodeA.Start()
	nodeB.Start()
	t.Logf("Node A started on %s", nodeAAddr.String())
	t.Logf("Node B started on %s", nodeBAddr.String())

	// Send some messages
	t.Log("=== Sending messages ===")

	// Node A sends ping to Node B
	err = nodeA.SendString(nodeBAddr, "ping", "Hello from A!")
	if err != nil {
		t.Errorf("Failed to send ping: %v", err)
	}

	// Node B sends ping to Node A
	err = nodeB.SendString(nodeAAddr, "ping", "Hello from B!")
	if err != nil {
		t.Errorf("Failed to send ping: %v", err)
	}

	// Wait for both ping messages to be received and pong responses with timeout
	waitForSignal(nodeBReceived, "Node B receives ping from A")
	waitForSignal(nodeAReceived, "Node A receives ping from B")
	waitForSignal(nodeAReceived, "Node A receives pong from B")
	waitForSignal(nodeBReceived, "Node B receives pong from A")

	// Clean up
	nodeA.Close()
	nodeB.Close()

	t.Log("Node communication test completed successfully")
}

func TestNetworkPartitioning(t *testing.T) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a mock network
	network := NewMockNetwork()

	// Define addresses
	nodeAAddr := Address{IP: "127.0.0.1", Port: 8080}
	nodeBAddr := Address{IP: "127.0.0.1", Port: 8081}

	// Create nodes
	nodeA, err := NewNode(network, nodeAAddr)
	if err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}

	nodeB, err := NewNode(network, nodeBAddr)
	if err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}

	// Channel for synchronization
	messageReceived := make(chan struct{}, 2)

	// Helper function to extract message content after ":"
	extractContent := func(payload []byte) string {
		payloadStr := string(payload)
		for i, char := range payloadStr {
			if char == ':' {
				return payloadStr[i+1:]
			}
		}
		return payloadStr
	}

	// Helper function to wait for channel with timeout
	waitForMessage := func(description string) {
		select {
		case <-messageReceived:
			t.Logf("✓ %s", description)
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for: %s", description)
		}
	}

	// Register message handler for node B
	nodeB.Handle("test", func(msg Message) error {
		content := extractContent(msg.Payload)
		t.Logf("Node B received message: %s", content)
		messageReceived <- struct{}{}
		return nil
	})

	// Start nodes
	nodeA.Start()
	nodeB.Start()

	// Test normal communication first
	t.Log("=== Testing normal communication ===")
	err = nodeA.SendString(nodeBAddr, "test", "Normal message")
	if err != nil {
		t.Errorf("Failed to send normal message: %v", err)
	}

	// Wait for message to be received with timeout
	waitForMessage("Normal message received")

	// Test network partitioning
	t.Log("=== Testing network partition ===")
	network.Partition([]Address{nodeAAddr}, []Address{nodeBAddr})

	err = nodeA.SendString(nodeBAddr, "test", "This should fail")
	if err == nil {
		t.Error("Expected partition error, but message was sent successfully")
	} else {
		t.Logf("✓ Expected partition error: %v", err)
	}

	// Heal the network
	t.Log("=== Testing network heal ===")
	network.Heal()

	err = nodeA.SendString(nodeBAddr, "test", "This should work again")
	if err != nil {
		t.Errorf("Unexpected error after heal: %v", err)
	}

	// Wait for final message to be received with timeout
	waitForMessage("Message after heal received")

	// Clean up
	nodeA.Close()
	nodeB.Close()

	t.Log("Network partitioning test completed successfully")
}

func TestBroadcastPattern(t *testing.T) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a mock network
	network := NewMockNetwork()

	// Create multiple nodes
	nodes := make([]*Node, 3)
	addrs := []Address{
		{IP: "127.0.0.1", Port: 8080},
		{IP: "127.0.0.1", Port: 8081},
		{IP: "127.0.0.1", Port: 8082},
	}

	// Channel to track received messages
	received := make(chan string, 10)

	// Helper function to extract message content after ":"
	extractContent := func(payload []byte) string {
		payloadStr := string(payload)
		for i, char := range payloadStr {
			if char == ':' {
				return payloadStr[i+1:]
			}
		}
		return payloadStr
	}

	// Helper function to wait for broadcast message with timeout
	waitForBroadcast := func(expectedCount int) {
		for i := 0; i < expectedCount; i++ {
			select {
			case msg := <-received:
				t.Logf("✓ %s", msg)
			case <-ctx.Done():
				t.Fatalf("Timeout waiting for broadcast message %d/%d", i+1, expectedCount)
			}
		}
	}

	// Create and start nodes
	for i, addr := range addrs {
		node, err := NewNode(network, addr)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node

		// Register broadcast handler
		node.Handle("broadcast", func(msg Message) error {
			content := extractContent(msg.Payload)
			received <- fmt.Sprintf("Node %s received: %s", msg.To.String(), content)
			return nil
		})

		node.Start()
	}

	// Node 0 broadcasts to all other nodes
	t.Log("=== Testing broadcast pattern ===")
	for i := 1; i < len(addrs); i++ {
		err := nodes[0].SendString(addrs[i], "broadcast", "Broadcast message")
		if err != nil {
			t.Errorf("Failed to send broadcast to node %d: %v", i, err)
		}
	}

	// Wait for all messages to be received with timeout
	expectedMessages := len(addrs) - 1 // All nodes except sender
	waitForBroadcast(expectedMessages)

	// Clean up
	for _, node := range nodes {
		node.Close()
	}

	t.Log("Broadcast pattern test completed successfully")
}

func TestRequestResponsePattern(t *testing.T) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a mock network
	network := NewMockNetwork()

	// Define addresses
	clientAddr := Address{IP: "127.0.0.1", Port: 8080}
	serverAddr := Address{IP: "127.0.0.1", Port: 8081}

	// Create nodes
	client, err := NewNode(network, clientAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	server, err := NewNode(network, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Channel for response synchronization
	responseReceived := make(chan string, 1)

	// Helper function to extract message content after ":"
	extractContent := func(payload []byte) string {
		payloadStr := string(payload)
		for i, char := range payloadStr {
			if char == ':' {
				return payloadStr[i+1:]
			}
		}
		return payloadStr
	}

	// Server handles requests
	server.Handle("request", func(msg Message) error {
		content := extractContent(msg.Payload)
		t.Logf("Server received request: %s", content)
		// Send response back
		return server.SendString(msg.From, "response", "Hello from server")
	})

	// Client handles responses
	client.Handle("response", func(msg Message) error {
		content := extractContent(msg.Payload)
		t.Logf("Client received response: %s", content)
		responseReceived <- content
		return nil
	})

	// Start nodes
	client.Start()
	server.Start()

	// Send request
	t.Log("=== Testing request-response pattern ===")
	err = client.SendString(serverAddr, "request", "Hello from client")
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}

	// Wait for response with timeout
	var response string
	select {
	case response = <-responseReceived:
		t.Logf("✓ Response received: %s", response)
	case <-ctx.Done():
		t.Fatal("Timeout waiting for response")
	}

	expected := "Hello from server"
	if response != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, response)
	}

	// Clean up
	client.Close()
	server.Close()

	t.Log("Request-response pattern test completed successfully")
}

func TestAllNodesCanPingEachOther(t *testing.T) {
	const numNodes = 5 // Change to desired number of nodes
	network := NewMockNetwork()
	nodes := make([]*Node, numNodes)
	addresses := make([]Address, numNodes)
	received := make([][]bool, numNodes)

	// Create nodes and addresses
	for i := 0; i < numNodes; i++ {
		addr := Address{IP: "127.0.0.1", Port: 8000 + i}
		node, err := NewNode(network, addr)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		addresses[i] = addr
		received[i] = make([]bool, numNodes)
	}

	// Each node handles pings and marks received
	for i, node := range nodes {
		idx := i
		node.Handle("ping", func(msg Message) error {
			fromIdx := -1
			for j, addr := range addresses {
				if addr == msg.From {
					fromIdx = j
					break
				}
			}
			if fromIdx >= 0 {
				received[idx][fromIdx] = true
			}
			return nil
		})
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	// Each node sends ping to every other node
	for i, node := range nodes {
		for j, addr := range addresses {
			if i != j {
				err := node.SendString(addr, "ping", "Hello!")
				if err != nil {
					t.Errorf("Node %d failed to ping node %d: %v", i, j, err)
				}
			}
		}
	}

	// Wait for messages to be processed
	time.Sleep(1 * time.Second)

	// Check that every node received a ping from every other node
	for i := 0; i < numNodes; i++ {
		for j := 0; j < numNodes; j++ {
			if i != j && !received[i][j] {
				t.Errorf("Node %d did not receive ping from node %d", i, j)
			}
		}
	}

	// Clean up
	for _, node := range nodes {
		node.Close()
	}
}
