package cli_test

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// Mock node interface for testing
type MockNode struct {
	store map[string][]byte
	addr  string
}

func NewMockNode(addr string) *MockNode {
	return &MockNode{
		store: make(map[string][]byte),
		addr:  addr,
	}
}

func (m *MockNode) StoreAtK(key string, value []byte, k int) error {
	m.store[key] = value
	return nil
}

func (m *MockNode) FindObject(key string) ([]byte, string, bool) {
	if value, found := m.store[key]; found {
		return value, m.addr, true
	}
	return nil, "", false
}

func (m *MockNode) FindObjectLocally(key string) ([]byte, bool) {
	value, found := m.store[key]
	return value, found
}

func (m *MockNode) Address() string {
	return m.addr
}

func TestHandlePut(t *testing.T) {
	// Create a mock node
	node := NewMockNode("127.0.0.1:8000")

	// Set as current node
	currentNode = node
	defer func() { currentNode = nil }()

	// Test put command
	testData := "test data"
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(testData)))

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	HandlePut(testData)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Should output the hash
	if !strings.Contains(outputStr, hash) {
		t.Errorf("Expected output to contain hash %s, got %s", hash, outputStr)
	}

	// Verify data was stored
	if value, found := node.FindObjectLocally(hash); found {
		if string(value) != testData {
			t.Errorf("Expected stored value %s, got %s", testData, string(value))
		}
	} else {
		t.Error("Data should be stored locally")
	}
}

func TestHandleGet(t *testing.T) {
	// Create a mock node
	node := NewMockNode("127.0.0.1:8000")

	// Set as current node
	currentNode = node
	defer func() { currentNode = nil }()

	// Store test data
	testData := "test data for get"
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(testData)))
	node.StoreAtK(hash, []byte(testData), 1)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	HandleGet(hash)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Check for the actual data content
	if !strings.Contains(outputStr, testData) {
		t.Errorf("Expected output to contain data '%s', got '%s'", testData, outputStr)
	}

	// Also check that source info is shown
	if !strings.Contains(outputStr, "Retrieved from node:") {
		t.Errorf("Expected output to show retrieval source, got '%s'", outputStr)
	}
}

func TestShowHelp(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showHelp()

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	expectedCommands := []string{"put", "get", "help", "exit"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(outputStr, cmd) {
			t.Errorf("Help output should contain command: %s", cmd)
		}
	}
}

func TestHandlePutWithNoNode(t *testing.T) {
	// Ensure no current node
	currentNode = nil

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	HandlePut("test data")

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Error: No node available") {
		t.Error("Should show error when no node available")
	}
}

func TestHandleGetWithNoNode(t *testing.T) {
	// Ensure no current node
	currentNode = nil

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	HandleGet("somehash")

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Error: No node available") {
		t.Error("Should show error when no node available")
	}
}

func TestHandleGetNotFound(t *testing.T) {
	// Create a mock node
	node := NewMockNode("127.0.0.1:8000")

	// Set as current node
	currentNode = node
	defer func() { currentNode = nil }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Try to get non-existent hash
	HandleGet("nonexistenthash")

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Object not found") {
		t.Errorf("Expected 'Object not found' message, got '%s'", outputStr)
	}
}

func TestPutGetWorkflow(t *testing.T) {
	// Create a mock node
	node := NewMockNode("127.0.0.1:8000")

	// Set as current node
	currentNode = node
	defer func() { currentNode = nil }()

	testData := "workflow test data"

	// Capture put output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	HandlePut(testData)

	w.Close()
	os.Stdout = oldStdout

	putOutput, _ := io.ReadAll(r)
	hash := strings.TrimSpace(string(putOutput))

	// Now try to get it back
	r, w, _ = os.Pipe()
	os.Stdout = w

	HandleGet(hash)

	w.Close()
	os.Stdout = oldStdout

	getOutput, _ := io.ReadAll(r)
	getOutputStr := string(getOutput)

	// Should contain the original data
	if !strings.Contains(getOutputStr, testData) {
		t.Errorf("Get should return original data '%s', got '%s'", testData, getOutputStr)
	}
}
