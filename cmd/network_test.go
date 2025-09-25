package main

import (
	"bytes"
	"testing"
	"time"
)

func TestAddressString(t *testing.T) {
	testCases := []struct {
		ip       string
		port     int
		expected string
	}{
		{"127.0.0.1", 8000, "127.0.0.1:8000"},
		{"localhost", 80, "localhost:80"},
		{"192.168.1.1", 1234, "192.168.1.1:1234"},
	}

	for _, tc := range testCases {
		addr := Address{
			IP:   tc.ip,
			Port: tc.port,
		}

		if addr.String() != tc.expected {
			t.Errorf("Expected %q, got %q", tc.expected, addr.String())
		}
	}
}

func TestMessageReplyString(t *testing.T) {
	// Create a mock network for testing
	network := NewMockNetwork()

	// Set up sender and receiver addresses
	senderAddr := Address{IP: "127.0.0.1", Port: 8001}
	receiverAddr := Address{IP: "127.0.0.1", Port: 8002}

	// First register the sender address as a listener (important!)
	senderConn, err := network.Listen(senderAddr)
	if err != nil {
		t.Fatalf("Failed to set up sender listener: %v", err)
	}
	defer senderConn.Close()

	// Create original message
	originalMsg := Message{
		From:    senderAddr,
		To:      receiverAddr,
		Payload: []byte("test-payload"),
		network: network,
	}

	// Set up a receiver to listen for the reply
	receivedReply := make(chan Message, 1)

	// Wait for and process the reply
	go func() {
		reply, err := senderConn.Recv()
		if err != nil {
			t.Errorf("Failed to receive reply: %v", err)
			return
		}

		receivedReply <- reply
	}()

	// Send the reply
	err = originalMsg.ReplyString("REPLY", "Hello back")
	if err != nil {
		t.Errorf("Failed to send reply: %v", err)
		return
	}

	// Wait for reply to be received
	select {
	case reply := <-receivedReply:
		if !bytes.Equal(reply.Payload, []byte("REPLY:Hello back")) {
			t.Errorf("Expected payload 'REPLY:Hello back', got '%s'", string(reply.Payload))
		}

		if reply.From != receiverAddr {
			t.Errorf("Reply sender should be receiver of original message")
		}

		if reply.To != senderAddr {
			t.Errorf("Reply recipient should be sender of original message")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for reply")
	}
}

func TestNetworkDialAndListen(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Define address for the listener
	listenerAddr := Address{IP: "127.0.0.1", Port: 8000}

	// Start listening
	listenerConn, err := network.Listen(listenerAddr)
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}

	// Message to be sent
	testMessage := Message{
		From:    Address{IP: "127.0.0.1", Port: 8001},
		To:      listenerAddr,
		Payload: []byte("test message"),
	}

	// Channel to collect received messages
	received := make(chan Message, 1)

	// Start receiving in a goroutine
	go func() {
		msg, err := listenerConn.Recv()
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
			return
		}
		received <- msg
	}()

	// Dial the listener
	senderConn, err := network.Dial(listenerAddr)
	if err != nil {
		t.Fatalf("Failed to dial listener: %v", err)
	}

	// Send the test message
	err = senderConn.Send(testMessage)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for message to be received
	select {
	case msg := <-received:
		if !bytes.Equal(msg.Payload, testMessage.Payload) {
			t.Errorf("Expected payload '%s', got '%s'", string(testMessage.Payload), string(msg.Payload))
		}
		if msg.From != testMessage.From {
			t.Error("Message sender doesn't match")
		}
		if msg.To != testMessage.To {
			t.Error("Message recipient doesn't match")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	// Clean up
	senderConn.Close()
	listenerConn.Close()
}

func TestNetworkPartitionAndHeal(t *testing.T) {
	// Create a mock network
	network := NewMockNetwork()

	// Define addresses for two groups
	group1 := []Address{
		{IP: "127.0.0.1", Port: 8001},
		{IP: "127.0.0.1", Port: 8002},
	}

	group2 := []Address{
		{IP: "127.0.0.1", Port: 8003},
		{IP: "127.0.0.1", Port: 8004},
	}

	// Create listeners for both groups
	listenerG1, err := network.Listen(group1[0])
	if err != nil {
		t.Fatalf("Failed to start listener for group1: %v", err)
	}
	defer listenerG1.Close()

	listenerG2, err := network.Listen(group2[0])
	if err != nil {
		t.Fatalf("Failed to start listener for group2: %v", err)
	}
	defer listenerG2.Close()

	// Create a partition between the groups
	network.Partition(group1, group2)

	// Try to send a message across the partition - this should fail
	testMsg := Message{
		From:    group1[0],
		To:      group2[0],
		Payload: []byte("test message"),
	}

	// We can dial, but the send should fail
	senderConn, err := network.Dial(group2[0])
	if err != nil {
		// It's also acceptable if Dial fails due to partitioning
		t.Logf("Dial failed due to partitioning: %v", err)
	} else {
		// If Dial succeeded, then Send should fail
		err = senderConn.Send(testMsg)
		if err == nil {
			t.Error("Expected send across partition to fail, but it succeeded")
		}
		senderConn.Close()
	}

	// Heal the network
	network.Heal()

	// Try to dial again after healing
	senderConn, err = network.Dial(group2[0])
	if err != nil {
		t.Fatalf("Failed to dial after healing: %v", err)
	}
	defer senderConn.Close()

	// Channel to collect received messages
	received := make(chan Message, 1)

	// Start receiving in a goroutine
	go func() {
		msg, err := listenerG2.Recv()
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
			return
		}
		received <- msg
	}()

	// Send the test message after healing
	err = senderConn.Send(testMsg)
	if err != nil {
		t.Fatalf("Failed to send message after healing: %v", err)
	}

	// Wait for message to be received
	select {
	case msg := <-received:
		if !bytes.Equal(msg.Payload, testMsg.Payload) {
			t.Errorf("Expected payload '%s', got '%s'", string(testMsg.Payload), string(msg.Payload))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message after healing")
	}
}

func TestSendPing(t *testing.T) {
	network := NewMockNetwork()

	fromAddr := Address{IP: "127.0.0.1", Port: 8001}
	toAddr := Address{IP: "127.0.0.1", Port: 8002}

	// Start a listener at the destination address
	conn, err := network.Listen(toAddr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	// Channel to collect received pings
	received := make(chan Message, 1)

	// Start receiving in a goroutine
	go func() {
		msg, err := conn.Recv()
		if err != nil {
			t.Errorf("Failed to receive ping: %v", err)
			return
		}
		received <- msg
	}()

	// Send ping
	err = SendPing(network, fromAddr, toAddr)
	if err != nil {
		t.Fatalf("SendPing failed: %v", err)
	}

	// Wait for ping to be received
	select {
	case msg := <-received:
		if !bytes.Equal(msg.Payload, []byte(MsgPing)) {
			t.Errorf("Expected payload '%s', got '%s'", MsgPing, string(msg.Payload))
		}
		if msg.From != fromAddr {
			t.Errorf("Expected sender %v, got %v", fromAddr, msg.From)
		}
		if msg.To != toAddr {
			t.Errorf("Expected recipient %v, got %v", toAddr, msg.To)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for ping")
	}

	conn.Close()
}

// TestMessageForwarding tests if a node can correctly forward messages to another node
func TestMessageForwarding(t *testing.T) {
	network := NewMockNetwork()

	// Set up three nodes: sender, forwarder, and receiver
	senderAddr := Address{IP: "127.0.0.1", Port: 8001}
	forwarderAddr := Address{IP: "127.0.0.1", Port: 8002}
	receiverAddr := Address{IP: "127.0.0.1", Port: 8003}

	// Create nodes
	sender, err := NewNode(network, senderAddr)
	if err != nil {
		t.Fatalf("Failed to create sender node: %v", err)
	}

	forwarder, err := NewNode(network, forwarderAddr)
	if err != nil {
		t.Fatalf("Failed to create forwarder node: %v", err)
	}

	receiver, err := NewNode(network, receiverAddr)
	if err != nil {
		t.Fatalf("Failed to create receiver node: %v", err)
	}

	// Set up message forwarding in the forwarder node
	forwarder.Handle("forward", func(msg Message) error {
		// Forward the message to the receiver
		return forwarder.SendString(receiverAddr, "forwarded", string(msg.Payload))
	})

	// Set up message receiving in the receiver node
	messageReceived := make(chan string, 1)
	receiver.Handle("forwarded", func(msg Message) error {
		messageReceived <- string(msg.Payload)
		return nil
	})

	// Start all nodes
	sender.Start()
	forwarder.Start()
	receiver.Start()

	// Send a message to be forwarded
	testMessage := "This is a test message to be forwarded"
	err = sender.SendString(forwarderAddr, "forward", testMessage)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Check if the message was received by the receiver
	select {
	case receivedMsg := <-messageReceived:
		if receivedMsg != testMessage {
			t.Errorf("Expected message '%s', got '%s'", testMessage, receivedMsg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for forwarded message")
	}

	// Clean up
	sender.Close()
	forwarder.Close()
	receiver.Close()
}
