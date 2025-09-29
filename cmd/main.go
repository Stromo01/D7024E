package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	// Parse command line flags
	var port = flag.Int("port", 8080, "Port for this node to listen on")
	var bootstrapIP = flag.String("bootstrap-ip", "", "IP address of bootstrap node")
	var bootstrapPort = flag.Int("bootstrap-port", 0, "Port of bootstrap node")
	var useRealNetwork = flag.Bool("real-network", false, "Use real UDP networking instead of mock")
	flag.Parse()

	fmt.Printf("🚀 Starting Kademlia DHT node on port %d\n", *port)

	var network Network
	if *useRealNetwork {
		fmt.Println("🌐 Using real UDP networking")
		network = NewUDPNetwork()
	} else {
		fmt.Println("🧪 Using mock network for testing")
		network = NewMockNetwork()
	}

	// Create local address
	localAddr := Address{IP: "127.0.0.1", Port: *port}

	// Create node
	node, err := NewNode(network, localAddr)
	if err != nil {
		fmt.Printf("❌ Failed to create node: %v\n", err)
		os.Exit(1)
	}

	// Start the node
	fmt.Printf("🔌 Creating node on %s:%d\n", localAddr.IP, localAddr.Port)
	node.Start()

	// If bootstrap node specified, join network
	if *bootstrapIP != "" && *bootstrapPort != 0 {
		bootstrapAddr := Address{IP: *bootstrapIP, Port: *bootstrapPort}
		fmt.Printf("🔗 Joining network via bootstrap node %s:%d\n", *bootstrapIP, *bootstrapPort)

		// Give the node a moment to start
		time.Sleep(100 * time.Millisecond)

		// Add bootstrap node to routing table
		contact := Triple{
			ID:   make([]byte, 20), // We don't know the bootstrap node's ID yet
			Addr: bootstrapAddr,
			Port: bootstrapAddr.Port,
		}
		// Try to ping the bootstrap node
		err := node.RPCPing(bootstrapAddr)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not ping bootstrap node: %v\n", err)
		} else {
			fmt.Println("✅ Successfully connected to bootstrap node")
			node.routing.addContact(contact)
		}
	} else {
		fmt.Println("🏗️  Starting as bootstrap node")
	}

	// Start interactive CLI
	fmt.Println("🎯 Node started successfully!")
	startKademliaCLI(node)
}
