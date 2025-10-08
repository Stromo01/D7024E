package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"
	"strconv"
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
		fmt.Printf("[%s] > ", node.addr.String())
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.SplitN(input, " ", 2)

		switch parts[0] {
		case "put":
			if len(parts) < 2 {
				fmt.Println("Usage: put <data>")
				continue
			}

			data := parts[1]
			hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))

			// Store at K closest nodes
			if err := node.StoreAtK(hash, []byte(data), K); err != nil {
				fmt.Printf("Error storing data: %v\n", err)
			} else {
				fmt.Printf("%s\n", hash)
			}

		case "get":
			if len(parts) < 2 {
				fmt.Println("Usage: get <hash>")
				continue
			}

			hash := parts[1]

			// First check local storage
			if value, found := node.FindObjectLocally(hash); found {
				fmt.Printf("%s\n", string(value))
				fmt.Printf("Retrieved from node: %s (local)\n", node.addr.String())
				continue
			}

			// If not found locally, search the network
			if value, sourceNode, found := node.FindObject(hash); found {
				fmt.Printf("%s\n", string(value))
				fmt.Printf("Retrieved from node: %s\n", sourceNode)
			} else {
				fmt.Println("Object not found")
			}

		case "exit":
			fmt.Println("Shutting down...")
			node.Close()
			return

		case "contacts":
			handleContactsCommand(node)

		case "ping":
			if len(parts) >= 2 {
				handlePingCommand(parts, node)
			} else {
				fmt.Println("Usage: ping <ip:port>")
			}

		default:
			fmt.Println("Unknown command. Available: put, get, contacts, ping, exit")
		}
	}
}

func handleContactsCommand(node *Node) {
	fmt.Println("Known contacts in routing table:")
	contacts := node.GetAllContacts()
	if len(contacts) == 0 {
		fmt.Println("  (no contacts)")
	} else {
		for i, contact := range contacts {
			if i >= 10 { // Limit output
				fmt.Printf("  ... and %d more\n", len(contacts)-10)
				break
			}
			fmt.Printf("  %s:%d\n", contact.Addr.IP, contact.Port)
		}
	}
}

func handlePingCommand(parts []string, node *Node) {
	target := parts[1]
	fmt.Printf("Pinging %s...\n", target)

	// Parse target address
	addrParts := strings.Split(target, ":")
	if len(addrParts) != 2 {
		fmt.Println("Invalid address format. Use ip:port")
		return
	}

	// Parse port
	port, err := strconv.Atoi(addrParts[1])
	if err != nil {
		fmt.Printf("Invalid port: %s\n", addrParts[1])
		return
	}

	targetAddr := Address{IP: addrParts[0], Port: port}

	// Send actual ping
	err = node.Send(targetAddr, MsgPing, []byte("ping"))
	if err != nil {
		fmt.Printf("Failed to ping %s: %v\n", target, err)
	} else {
		fmt.Printf("Ping sent to %s successfully\n", target)
	}
}
