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

func TestNodeMessageHandling(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Test handler registration
	called := false
	node.Handle("test", func(msg Message) error {
		called = true
		return nil
	})

	go node.Start()

	// Send message to self
	err = node.Send(node.Address(), "test", []byte("data"))
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if !called {
		t.Error("Handler should have been called")
	}
}

func TestNodeGetAllContacts(t *testing.T) {
	network := NewUDPNetwork()
	addr := Address{IP: "127.0.0.1", Port: 0}

	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	contactAddr := Address{IP: "127.0.0.1", Port: 0}
	contactNode, err := NewNode(network, contactAddr)
	if err != nil {
		t.Fatalf("Failed to create contact node: %v", err)
	}
	defer contactNode.Close()

	go node.Start()
	go contactNode.Start()

	time.Sleep(50 * time.Millisecond) // Let nodes start

	// Create proper message with FromContact
	msg := Message{
		From: contactNode.Address(),
		To:   node.Address(),
		FromContact: Triple{
			ID:   contactNode.Id[:],
			Addr: contactNode.Address(),
			Port: contactNode.Address().Port,
		},
		Payload: []byte(MsgPing + ":ping"),
		Network: network,
	}

	// Send via connection instead of node.Send to include FromContact
	conn, err := network.Dial(node.Address())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	err = conn.Send(msg)
	if err != nil {
		t.Errorf("Failed to send ping: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	contacts := node.GetAllContacts()
	if len(contacts) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(contacts))
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

func TestNodeSend(t *testing.T) {
	network := NewUDPNetwork()
	senderAddr := Address{IP: "127.0.0.1", Port: 0}
	receiverAddr := Address{IP: "127.0.0.1", Port: 0}

	sender, err := NewNode(network, senderAddr)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	receiver, err := NewNode(network, receiverAddr)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Close()

	// Set up receiver
	received := make(chan bool, 1)
	receiver.Handle("ping", func(msg Message) error {
		received <- true
		return nil
	})

	go sender.Start()
	go receiver.Start()

	// Send message
	err = sender.Send(receiver.Address(), "ping", []byte("hello"))
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}

	select {
	case <-received:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("Message not received")
	}
}

func TestNodePingHandler(t *testing.T) {
	network := NewUDPNetwork()
	nodeAAddr := Address{IP: "127.0.0.1", Port: 0}
	nodeBAddr := Address{IP: "127.0.0.1", Port: 0}

	nodeA, err := NewNode(network, nodeAAddr)
	if err != nil {
		t.Fatalf("Failed to create nodeA: %v", err)
	}
	defer nodeA.Close()

	nodeB, err := NewNode(network, nodeBAddr)
	if err != nil {
		t.Fatalf("Failed to create nodeB: %v", err)
	}
	defer nodeB.Close()

	go nodeA.Start()
	go nodeB.Start()

	time.Sleep(50 * time.Millisecond) // Let nodes start properly

	// Create proper ping message
	msg := Message{
		From: nodeA.Address(),
		To:   nodeB.Address(),
		FromContact: Triple{
			ID:   nodeA.Id[:],
			Addr: nodeA.Address(),
			Port: nodeA.Address().Port,
		},
		Payload: []byte(MsgPing + ":ping"),
		Network: network,
	}

	conn, err := network.Dial(nodeB.Address())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	err = conn.Send(msg)
	if err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	time.Sleep(200 * time.Millisecond) // Longer wait

	contacts := nodeB.GetAllContacts()
	found := false
	for _, contact := range contacts {
		if bytes.Equal(contact.ID, nodeA.Id[:]) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Sender should be added to receiver's routing table")
	}
}

func TestNodeJoinNetwork(t *testing.T) {
	network := NewUDPNetwork()
	bootstrapAddr := Address{IP: "127.0.0.1", Port: 0}
	joinerAddr := Address{IP: "127.0.0.1", Port: 0}

	bootstrap, err := NewNode(network, bootstrapAddr)
	if err != nil {
		t.Fatalf("Failed to create bootstrap: %v", err)
	}
	defer bootstrap.Close()

	joiner, err := NewNode(network, joinerAddr)
	if err != nil {
		t.Fatalf("Failed to create joiner: %v", err)
	}
	defer joiner.Close()

	go bootstrap.Start()
	go joiner.Start()

	// Join network
	bootstrapTriple := Triple{
		ID:   bootstrap.Id[:],
		Addr: bootstrap.Address(),
		Port: bootstrap.Address().Port,
	}

	err = joiner.JoinNetwork(bootstrapTriple)
	if err != nil {
		t.Errorf("JoinNetwork failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Check bootstrap is in joiner's contacts
	contacts := joiner.GetAllContacts()
	found := false
	for _, contact := range contacts {
		if bytes.Equal(contact.ID, bootstrap.Id[:]) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Bootstrap node should be in joiner's routing table")
	}
}
