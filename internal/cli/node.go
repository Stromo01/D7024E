package cli

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// We'll need to import the main package for Kademlia types
// This will be addressed when we fix the module structure

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Start a Kademlia DHT node",
	Long:  "Start a Kademlia DHT node with interactive CLI commands",
	Run:   runNode,
}

var (
	port          int
	bootstrapIP   string
	bootstrapPort int
	// Add global node variable to hold the actual Kademlia node
	kademliaNode interface{} // Will be *Node when properly imported
)

func init() {
	nodeCmd.Flags().IntVar(&port, "port", 8080, "Port for this node to listen on")
	nodeCmd.Flags().StringVar(&bootstrapIP, "bootstrap-ip", "", "IP address of bootstrap node")
	nodeCmd.Flags().IntVar(&bootstrapPort, "bootstrap-port", 0, "Port of bootstrap node")

	rootCmd.AddCommand(nodeCmd)
}

func runNode(cmd *cobra.Command, args []string) {
	fmt.Printf("Starting Kademlia DHT node on port %d\n", port)

	// TODO: This will need to be implemented with real networking
	// For now, this shows the structure needed

	// Create mock network for demonstration
	// In real implementation, this would be UDPNetwork
	fmt.Println("Creating network...")

	// Create node
	fmt.Printf("Creating node on 127.0.0.1:%d\n", port)

	// If bootstrap node specified, join network
	if bootstrapIP != "" && bootstrapPort != 0 {
		fmt.Printf("Joining network via bootstrap node %s:%d\n", bootstrapIP, bootstrapPort)
		// TODO: Implement network joining
	} else {
		fmt.Println("Starting as bootstrap node")
	}

	// Start interactive CLI
	startInteractiveCLI()
}

func startInteractiveCLI() {
	fmt.Println("\nKademlia DHT Node CLI")
	fmt.Println("Available commands:")
	fmt.Println("  put <data>     - Store data and return hash")
	fmt.Println("  get <hash>     - Retrieve data by hash")
	fmt.Println("  exit           - Exit the node")
	fmt.Println()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down node...")
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
			handlePutCommand(parts)
		case "get":
			handleGetCommand(parts)
		case "exit":
			fmt.Println("Exiting...")
			return
		case "help":
			showHelp()
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}
	}
}

func handlePutCommand(parts []string) {
	if len(parts) < 2 {
		fmt.Println("Usage: put <data>")
		return
	}

	// Join all parts after "put" as the data
	data := strings.Join(parts[1:], " ")

	// Calculate SHA-1 hash (matching Kademlia spec)
	hash := sha1.Sum([]byte(data))
	hashStr := fmt.Sprintf("%x", hash)

	fmt.Printf("Storing data: '%s'\n", data)

	// TODO: Implement actual storage using IterativeStore
	// For now, simulate storage
	fmt.Printf("Data stored successfully!\n")
	fmt.Printf("Hash: %s\n", hashStr)

	// In real implementation:
	// err := node.IterativeStore(hashStr, []byte(data))
	// if err != nil {
	//     fmt.Printf("Failed to store data: %v\n", err)
	// } else {
	//     fmt.Printf("Hash: %s\n", hashStr)
	// }
}

func handleGetCommand(parts []string) {
	if len(parts) != 2 {
		fmt.Println("Usage: get <hash>")
		return
	}

	hash := parts[1]

	// Validate hash format (should be 40 hex characters for SHA-1)
	if len(hash) != 40 {
		fmt.Println("Invalid hash format. Hash should be 40 hexadecimal characters.")
		return
	}

	fmt.Printf("Searching for data with hash: %s\n", hash)

	// TODO: Implement actual retrieval using IterativeFindValue
	// For now, simulate retrieval
	fmt.Printf("Data retrieved from node 127.0.0.1:8081\n")
	fmt.Printf("Content: example data for hash %s\n", hash)

	// In real implementation:
	// value, nodes, found := node.IterativeFindValue(hash)
	// if found {
	//     fmt.Printf("Content: %s\n", string(value))
	//     fmt.Printf("Retrieved from network\n")
	// } else {
	//     fmt.Printf("Data not found. Searched %d nodes.\n", len(nodes))
	// }
}

func showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  put <data>     - Store data in the DHT and return its hash")
	fmt.Println("  get <hash>     - Retrieve data from the DHT using its hash")
	fmt.Println("  exit           - Exit the node")
	fmt.Println("  help           - Show this help message")
}
