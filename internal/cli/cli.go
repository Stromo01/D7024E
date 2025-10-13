package cli

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"
)

func StartInteractiveCLI(node interface{}) {
	currentNode = node
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Kademlia CLI started. Type 'help' for commands or 'exit' to quit.")
	fmt.Print("kademlia> ")

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Print("kademlia> ")
			continue
		}

		args := strings.Fields(input)
		if len(args) == 0 {
			fmt.Print("kademlia> ")
			continue
		}

		command := args[0]

		switch command {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "help":
			showHelp()
		case "put":
			if len(args) < 2 {
				fmt.Println("Usage: put <data>")
			} else {
				handlePut(strings.Join(args[1:], " "))
			}
		case "get":
			if len(args) < 2 {
				fmt.Println("Usage: get <hash>")
			} else {
				handleGet(args[1])
			}
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}

		fmt.Print("kademlia> ")
	}
}

func showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  put <data>  - Store data and return hash")
	fmt.Println("  get <hash>  - Retrieve data by hash")
	fmt.Println("  help        - Show this help")
	fmt.Println("  exit        - Exit the CLI")
}

func handlePut(data string) {
	if currentNode == nil {
		fmt.Println("Error: No node available")
		return
	}

	hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))

	if node, ok := currentNode.(interface{ iterativeStore(string, []byte) }); ok {
		node.iterativeStore(hash, []byte(data))
		fmt.Printf("%s\n", hash)
	} else {
		fmt.Println("Error: Node does not support iterativeStore method")
	}
}

func handleGet(hash string) {
	if currentNode == nil {
		fmt.Println("Error: No node available")
		return
	}

	// You'll need to implement this based on your node's get method
	fmt.Printf("Getting data for hash: %s\n", hash)
	// Example: if node, ok := currentNode.(interface{ FindObject(string) ([]byte, string, bool) }); ok {
	//     value, source, found := node.FindObject(hash)
	//     if found {
	//         fmt.Printf("Found: %s (from %s)\n", string(value), source)
	//     } else {
	//         fmt.Printf("Not found\n")
	//     }
	// }
}
