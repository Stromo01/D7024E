package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/eislab-cps/go-template/internal/cli"
	"github.com/eislab-cps/go-template/pkg/build"
)

var (
	BuildVersion string = ""
	BuildTime    string = ""
)

func main() {
	build.BuildVersion = BuildVersion
	build.BuildTime = BuildTime

	// Check if we should run CLI commands or start node directly
	if len(os.Args) > 1 && (os.Args[1] == "rpc" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		// Use CLI framework for rpc command or help
		cli.Execute()
		return
	}

	// Otherwise, start a real node
	startKademliaNode()
}

func startKademliaNode() {
	// Command line flags
	isBootstrapPtr := flag.Bool("bootstrap", false, "If true, start as bootstrap node")
	portPtr := flag.Int("port", 8000, "Port to listen on")
	bootstrapAddrPtr := flag.String("bootstrap-addr", "127.0.0.1:8000", "Bootstrap node address (ip:port)")

	flag.Parse()

	// Create UDP network
	network := NewUDPNetwork()

	// Use localhost for simplicity
	ip := "127.0.0.1"

	// Create local address
	addr := Address{
		IP:   ip,
		Port: *portPtr,
	}

	// Create node
	node, err := NewNode(network, addr)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	// Start the node
	go node.Start()

	if *isBootstrapPtr {
		// Start as bootstrap node
		log.Printf("Starting as bootstrap node on %s", addr.String())
	} else {
		// Join existing network
		log.Printf("Starting node on %s, joining via %s", addr.String(), *bootstrapAddrPtr)

		// Parse bootstrap address
		parts := strings.Split(*bootstrapAddrPtr, ":")
		if len(parts) == 2 {
			bootPort, err := strconv.Atoi(parts[1])
			if err != nil {
				log.Fatalf("Invalid bootstrap port: %v", err)
			}

			bootAddr := Address{IP: parts[0], Port: bootPort}
			bootTriple := Triple{
				ID:   make([]byte, 20), // Proper 20-byte ID (will be updated when we get PONG)
				Addr: bootAddr,
				Port: bootPort,
			}

			if err := node.JoinNetwork(bootTriple); err != nil {
				log.Printf("Failed to join network: %v", err)
			} else {
				log.Printf("Successfully joined network via %s", *bootstrapAddrPtr)
			}
		} else {
			log.Printf("Invalid bootstrap address format: %s", *bootstrapAddrPtr)
		}
	}

	// Start CLI loop
	fmt.Printf("Node %s started successfully\n", addr.String())
	StartInteractiveNode(node)
}
