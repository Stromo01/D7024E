package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

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
	if len(os.Args) > 1 && (os.Args[1] == "rpc" || os.Args[1] == "put" || os.Args[1] == "get") {
		// Use CLI framework for commands
		cli.Execute()
		return
	}

	// Otherwise, start a real node
	startKademliaNode()
}

func startKademliaNode() {
	// Command line flags
	portPtr := flag.Int("port", 8000, "Port to listen on")
	bootstrapAddrPtr := flag.String("bootstrap-addr", "127.0.0.1:8000", "Bootstrap node address (ip:port)")
	isBootstrapPtr := flag.Bool("bootstrap", false, "Run as bootstrap node")

	flag.Parse()

	// Override with environment variables if they exist (for Docker)
	if portEnv := os.Getenv("NODE_PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			*portPtr = p
		}
	}

	if os.Getenv("IS_BOOTSTRAP") == "true" {
		*isBootstrapPtr = true
	}

	if bootstrapEnv := os.Getenv("BOOTSTRAP_ADDR"); bootstrapEnv != "" {
		*bootstrapAddrPtr = bootstrapEnv
	}

	// Debug logging
	log.Printf("Port: %d, IsBootstrap: %t, BootstrapAddr: %s", *portPtr, *isBootstrapPtr, *bootstrapAddrPtr)

	// Create UDP network
	network := NewUDPNetwork()

	// Get advertise address (what other nodes use to contact us)
	advertiseHost := os.Getenv("ADVERTISE_HOST")
	if advertiseHost == "" {
		// In Docker, use container hostname
		if hostname, err := os.Hostname(); err == nil {
			advertiseHost = hostname
		} else {
			advertiseHost = "127.0.0.1" // fallback for local dev
		}
	}

	// Create the address we'll advertise to other nodes
	advertiseAddr := Address{IP: advertiseHost, Port: *portPtr}

	// Debug logging
	log.Printf("Creating node with advertise address: %s", advertiseAddr.String())

	// Create node - it will handle listening internally
	node, err := NewNode(network, advertiseAddr)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	// Start the node in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		node.Start()
	}()

	if *isBootstrapPtr {
		log.Printf("Starting as bootstrap node on %s", advertiseAddr.String())
	} else {
		log.Printf("Starting node on %s, attempting to join via %s", advertiseAddr.String(), *bootstrapAddrPtr)
		parts := strings.Split(*bootstrapAddrPtr, ":")
		if len(parts) == 2 {
			bootPort, err := strconv.Atoi(parts[1])
			if err != nil {
				log.Printf("Invalid bootstrap port: %v", err)
			} else {
				bootAddr := Address{IP: parts[0], Port: bootPort}
				bootTriple := Triple{
					ID:   make([]byte, 20), // placeholder; JoinNetwork/SendPing will populate/update
					Addr: bootAddr,
					Port: bootPort,
				}

				if err := node.JoinNetwork(bootTriple); err != nil {
					log.Printf("Failed to join network: %v", err)
				} else {
					log.Printf("Successfully joined network via %s", *bootstrapAddrPtr)
				}
			}
		} else {
			log.Printf("Invalid bootstrap address format: %s", *bootstrapAddrPtr)
		}
	}

	// Start interactive CLI (blocks until user exits)
	fmt.Printf("Node %s started successfully\n", advertiseAddr.String())

	// Check if running in Docker (non-interactive environment)
	if os.Getenv("DOCKER_ENV") == "true" {
		log.Printf("Running in Docker mode - keeping node alive without interactive CLI")
		// Keep the node running without interactive CLI
		select {} // Block forever
	} else {
		// Only start interactive CLI if we're in a terminal
		StartInteractiveNode(node)
	}

	// When interactive loop ends, stop node and wait for goroutine to finish
	if closer, ok := interface{}(node).(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	wg.Wait()
}
