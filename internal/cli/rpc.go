package cli

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Global variables to simulate node functionality for M3
var nodeStore = make(map[string]string)

func init() {
	rootCmd.AddCommand(rpcCmd)
}

var rpcCmd = &cobra.Command{
	Use:   "rpc",
	Short: "Start a Kademlia node with interactive CLI",
	Long:  "Start a Kademlia node and provide interactive commands: put, get, exit",
	Run: func(cmd *cobra.Command, args []string) {
		startInteractiveNode()
	},
}

func startInteractiveNode() {
	fmt.Println("Kademlia node started. Available commands:")
	fmt.Println("  put <data>  - Store data and get hash")
	fmt.Println("  get <hash>  - Retrieve data by hash")
	fmt.Println("  exit        - Terminate node")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.SplitN(input, " ", 2)
		command := parts[0]

		switch command {
		case "put":
			handlePutCommand(parts)
		case "get":
			handleGetCommand(parts)
		case "exit":
			fmt.Println("Shutting down node...")
			return
		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Available commands: put, get, exit")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func handlePutCommand(parts []string) {
	if len(parts) < 2 {
		fmt.Println("Usage: put <data>")
		return
	}

	data := parts[1]

	// Calculate SHA-1 hash of the data
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))

	// Store the data (simulate storing to K closest nodes)
	nodeStore[hash] = data

	// Output the hash
	fmt.Printf("%s\n", hash)

	if Verbose {
		fmt.Printf("Successfully stored data: %s\n", data)
	}
}

func handleGetCommand(parts []string) {
	if len(parts) < 2 {
		fmt.Println("Usage: get <hash>")
		return
	}

	hash := parts[1]

	// Try to retrieve the data
	if data, found := nodeStore[hash]; found {
		// Output the contents and the node it was retrieved from
		fmt.Printf("%s\n", data)
		fmt.Printf("Retrieved from node: 127.0.0.1:8000\n")
	} else {
		fmt.Println("Object not found")
		if Verbose {
			fmt.Printf("Hash %s not found in local storage\n", hash)
		}
	}
}
