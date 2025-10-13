package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var currentNode interface{}

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
				HandlePut(strings.Join(args[1:], " "))
			}
		case "get":
			if len(args) < 2 {
				fmt.Println("Usage: get <hash>")
			} else {
				HandleGet(args[1])
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
