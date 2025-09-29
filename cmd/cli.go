package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Global node variable to hold the running Kademlia node
var runningNode *Node
var nodeNetwork Network

// startKademliaCLI starts the interactive CLI for Kademlia operations
func startKademliaCLI(node *Node) {
	runningNode = node

	fmt.Println("\n🚀 Kademlia DHT Node CLI")
	fmt.Println("Available commands:")
	fmt.Println("  put <data>     - Store data and return hash")
	fmt.Println("  get <hash>     - Retrieve data by hash")
	fmt.Println("  exit           - Exit the node")
	fmt.Println("  help           - Show help")
	fmt.Println()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n🔌 Shutting down node...")
		if runningNode != nil {
			runningNode.Close()
		}
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("kademlia> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "put":
			handleCLIPutCommand(parts)
		case "get":
			handleCLIGetCommand(parts)
		case "exit":
			fmt.Println("👋 Exiting...")
			if runningNode != nil {
				runningNode.Close()
			}
			return
		case "help":
			showCLIHelp()
		default:
			fmt.Printf("❌ Unknown command: %s. Type 'help' for available commands.\n", command)
		}
	}
}

func handleCLIPutCommand(parts []string) {
	if len(parts) < 2 {
		fmt.Println("❌ Usage: put <data>")
		return
	}

	// Join all parts after "put" as the data
	data := strings.Join(parts[1:], " ")

	// Calculate SHA-1 hash (matching Kademlia spec)
	hash := sha1.Sum([]byte(data))
	hashStr := fmt.Sprintf("%x", hash)

	fmt.Printf("📦 Storing data: '%s'\n", data)

	if runningNode == nil {
		fmt.Println("❌ Node not initialized")
		return
	}

	// Use actual IterativeStore
	err := runningNode.IterativeStore(hashStr, []byte(data))
	if err != nil {
		fmt.Printf("❌ Failed to store data: %v\n", err)
	} else {
		fmt.Printf("✅ Data stored successfully!\n")
		fmt.Printf("🔑 Hash: %s\n", hashStr)
	}
}

func handleCLIGetCommand(parts []string) {
	if len(parts) != 2 {
		fmt.Println("❌ Usage: get <hash>")
		return
	}

	hash := parts[1]

	// Validate hash format (should be 40 hex characters for SHA-1)
	if len(hash) != 40 {
		fmt.Println("❌ Invalid hash format. Hash should be 40 hexadecimal characters.")
		return
	}

	fmt.Printf("🔍 Searching for data with hash: %s\n", hash)

	if runningNode == nil {
		fmt.Println("❌ Node not initialized")
		return
	}

	// Use actual IterativeFindValue
	value, nodes, found := runningNode.IterativeFindValue(hash)
	if found {
		fmt.Printf("✅ Content: %s\n", string(value))
		fmt.Printf("📍 Retrieved from network (searched %d nodes)\n", len(nodes))
	} else {
		fmt.Printf("❌ Data not found. Searched %d nodes.\n", len(nodes))
	}
}

func showCLIHelp() {
	fmt.Println("📚 Available commands:")
	fmt.Println("  put <data>     - Store data in the DHT and return its hash")
	fmt.Println("  get <hash>     - Retrieve data from the DHT using its hash")
	fmt.Println("  exit           - Exit the node")
	fmt.Println("  help           - Show this help message")
}
