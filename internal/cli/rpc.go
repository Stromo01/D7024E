package cli

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Global store for simulation
var simulatedStore = make(map[string]string)

var rpcCmd = &cobra.Command{
	Use:   "rpc",
	Short: "Start a simulated Kademlia node (for testing)",
	Long:  "Start a simulated Kademlia node for testing basic CLI functionality",
	Run: func(cmd *cobra.Command, args []string) {
		startSimulatedNode()
	},
}

func init() {
	rootCmd.AddCommand(rpcCmd)
}

func startSimulatedNode() {
	fmt.Println("Starting simulated Kademlia node...")
	fmt.Println("Available commands:")
	fmt.Println("  put <data>  - Store data and get hash")
	fmt.Println("  get <hash>  - Retrieve data by hash")
	fmt.Println("  exit        - Terminate node")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("[simulated] > ")
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
			if len(parts) < 2 {
				fmt.Println("Usage: put <data>")
				continue
			}

			data := parts[1]
			hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))
			simulatedStore[hash] = data
			fmt.Printf("%s\n", hash)

		case "get":
			if len(parts) < 2 {
				fmt.Println("Usage: get <hash>")
				continue
			}

			hash := parts[1]
			if data, found := simulatedStore[hash]; found {
				fmt.Printf("%s\n", data)
				fmt.Printf("Retrieved from node: simulated\n")
			} else {
				fmt.Println("Object not found")
			}

		case "exit":
			fmt.Println("Shutting down simulated node...")
			return

		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Available commands: put, get, exit")
		}
	}
}
