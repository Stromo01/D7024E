package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestStartInteractiveNode_PutCommand(t *testing.T) {
	// Create a mock network and node
	network := NewMockNetwork()
	addr := Address{IP: "127.0.0.1", Port: 8000}
	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Start the node
	node.Start()

	// Test that StoreAtK works
	testData := "test data"
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(testData)))

	err = node.StoreAtK(hash, []byte(testData), K)
	if err != nil {
		t.Errorf("StoreAtK failed: %v", err)
	}

	// Verify data was stored locally
	value, found := node.FindObjectLocally(hash)
	if !found {
		t.Error("Data should be stored locally")
	}

	if string(value) != testData {
		t.Errorf("Expected %s, got %s", testData, string(value))
	}
}

func TestStartInteractiveNode_GetCommand(t *testing.T) {
	// Create a mock network and node
	network := NewMockNetwork()
	addr := Address{IP: "127.0.0.1", Port: 8000}
	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Start the node
	node.Start()

	// Store test data
	testData := "test data for get"
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(testData)))
	node.StoreObject(hash, []byte(testData))

	// Test FindObjectLocally
	value, found := node.FindObjectLocally(hash)
	if !found {
		t.Error("Should find object locally")
	}

	if string(value) != testData {
		t.Errorf("Expected %s, got %s", testData, string(value))
	}

	// Test FindObject (which includes network search)
	value, source, found := node.FindObject(hash)
	if !found {
		t.Error("Should find object")
	}

	if string(value) != testData {
		t.Errorf("Expected %s, got %s", testData, string(value))
	}

	if source != addr.String() {
		t.Errorf("Expected source %s, got %s", addr.String(), source)
	}
}

func TestHandleContactsCommand(t *testing.T) {
	// Create a mock network and node
	network := NewMockNetwork()
	addr := Address{IP: "127.0.0.1", Port: 8000}
	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Add some contacts
	contact1 := Triple{
		ID:   []byte("contact1"),
		Addr: Address{IP: "127.0.0.1", Port: 8001},
		Port: 8001,
	}
	contact2 := Triple{
		ID:   []byte("contact2"),
		Addr: Address{IP: "127.0.0.1", Port: 8002},
		Port: 8002,
	}

	node.routing.addContact(contact1)
	node.routing.addContact(contact2)

	// Test GetAllContacts
	contacts := node.GetAllContacts()
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}

	// Capture stdout for testing handleContactsCommand
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handleContactsCommand(node)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Known contacts") {
		t.Error("Output should contain 'Known contacts'")
	}
}

func TestHandlePingCommand(t *testing.T) {
	// Create a mock network and node
	network := NewMockNetwork()
	addr := Address{IP: "127.0.0.1", Port: 8000}
	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	node.Start()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test valid ping command
	parts := []string{"ping", "127.0.0.1:8001"}
	handlePingCommand(parts, node)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Pinging") {
		t.Error("Output should contain 'Pinging'")
	}
}

func TestHandlePingCommand_InvalidFormat(t *testing.T) {
	network := NewMockNetwork()
	addr := Address{IP: "127.0.0.1", Port: 8000}
	node, err := NewNode(network, addr)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Close()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test invalid ping command
	parts := []string{"ping", "invalid-address"}
	handlePingCommand(parts, node)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Invalid address format") {
		t.Error("Output should contain 'Invalid address format'")
	}
}
