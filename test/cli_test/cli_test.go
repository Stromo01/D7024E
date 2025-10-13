package cli_test

import (
	"io"
	"os"
	"strings"
	"testing"
)

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

	// Check for usage examples
	if !strings.Contains(outputStr, "put <data>") {
		t.Error("Help should show put usage")
	}
	if !strings.Contains(outputStr, "get <hash>") {
		t.Error("Help should show get usage")
	}
}

func TestCurrentNodeVariable(t *testing.T) {
	// Test that currentNode can be set
	testNode := "test-node"
	currentNode = testNode

	if currentNode != testNode {
		t.Error("currentNode should be settable")
	}

	// Clean up
	currentNode = nil

	if currentNode != nil {
		t.Error("currentNode should be nil after cleanup")
	}
}
