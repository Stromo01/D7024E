package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"
)

func StartInteractiveNode(node *Node) {
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
		parts := strings.SplitN(input, " ", 2)

		switch parts[0] {
		case "put":
			if len(parts) < 2 {
				fmt.Println("Usage: put <data>")
				continue
			}

			data := parts[1]
			hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))
			node.StoreAtK(hash, []byte(data), K)
			fmt.Printf("Stored with hash: %s\n", hash)

		case "get":
			if len(parts) < 2 {
				fmt.Println("Usage: get <hash>")
				continue
			}

			hash := parts[1]
			if value, found := node.FindObject(hash); found {
				fmt.Printf("Found: %s\n", string(value))
				fmt.Printf("Retrieved from: %s\n", node.addr.String())
			} else {
				fmt.Println("Object not found")
			}

		case "exit":
			fmt.Println("Shutting down...")
			node.Close()
			return

		default:
			fmt.Println("Unknown command")
		}
	}
}
