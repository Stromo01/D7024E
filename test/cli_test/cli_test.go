// Add to test/cli_test/cli_test.go
package cli_test

import (
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/eislab-cps/go-template/internal/cli"
)

// Mock node for testing
type MockNode struct {
	storedData map[string][]byte
}

// Implement Close() error to satisfy interface
func (m *MockNode) Close() error {
	return nil
}

func (m *MockNode) IterativeStore(hash string, data []byte) {
	if m.storedData == nil {
		m.storedData = make(map[string][]byte)
	}
	m.storedData[hash] = data
}

func (m *MockNode) FindObject(hash string) ([]byte, string, bool) {
	if data, exists := m.storedData[hash]; exists {
		return data, "local", true
	}
	return nil, "", false
}

func TestHandlePut(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up mock node
	mock := &MockNode{}
	CurrentNode = mock

	// Test HandlePut
	testData := "test data"
	HandlePut(testData)

	// Restore stdout and read output
	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Fix: Use the correct SHA-1 hash for "test data"
	// SHA-1 of "test data" is: f48dd853820860816c75d54d0f584dc863327a7c
	expectedHash := "f48dd853820860816c75d54d0f584dc863327a7c"
	if !strings.Contains(outputStr, expectedHash) {
		t.Errorf("HandlePut should output correct hash for test data. Expected: %s, Got: %s", expectedHash, outputStr)
	}
}

func TestHandlePutNoNode(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Clear current node
	CurrentNode = nil

	HandlePut("test")

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)

	if !strings.Contains(string(output), "Error: No node available") {
		t.Error("Should show error when no node available")
	}
}

func TestHandleGet(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up mock node with data
	mock := &MockNode{
		storedData: map[string][]byte{
			"testhash": []byte("test data"),
		},
	}
	CurrentNode = mock

	HandleGet("testhash")

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "test data") {
		t.Error("HandleGet should display found data")
	}
	if !strings.Contains(outputStr, "local") {
		t.Error("HandleGet should display source")
	}
}

func TestHandleGetNotFound(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	mock := &MockNode{}
	CurrentNode = mock

	HandleGet("nonexistent")

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)

	if !strings.Contains(string(output), "Error: Data not found") {
		t.Error("Should show error when data not found")
	}
}

func TestStartInteractiveCLI(t *testing.T) {
	// Create a pipe to simulate user input
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Set up mock node
	mock := &MockNode{}

	// Write commands and close input
	go func() {
		w.WriteString("help\n")
		w.WriteString("exit\n")
		w.Close()
	}()

	// Run CLI
	StartInteractiveCLI(mock)

	// Restore and read output
	os.Stdin = oldStdin
	wOut.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(rOut)

	outputStr := string(output)
	if !strings.Contains(outputStr, "Kademlia CLI started") {
		t.Error("Should show CLI start message")
	}
	if !strings.Contains(outputStr, "Available commands") {
		t.Error("Should show help when help command is used")
	}
}
