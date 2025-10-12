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
	portPtr := flag.Int("port", 8000, "Port to listen on")
	bootstrapAddrPtr := flag.String("bootstrap-addr", "127.0.0.1:8000", "Bootstrap node address (ip:port)")
	isBootstrapPtr := flag.Bool("bootstrap", false, "Run as bootstrap node")

	flag.Parse()

	// Create UDP network
	network := NewUDPNetwork()

	// bind listener to all interfaces so other containers/hosts can reach us
	listenIP := "0.0.0.0"
	listenAddr := Address{IP: listenIP, Port: *portPtr}
	listenerConn, err := network.Listen(listenAddr)
	if err != nil {
		log.Fatalf("failed to bind listener on %s: %v", listenAddr.String(), err)
	}
	// keep the listening connection open for the lifetime of the node
	// it will be closed when the program exits or when you explicitly Close() it
	defer listenerConn.Close()

	// Advertised IP/host that other nodes should use to contact this node.
	// Allow override for docker with ADVERTISE_HOST env var (e.g. "bootstrap" or container IP).
	// If ADVERTISE_HOST is not set we advertise the container's IP (or loopback for local dev).
	advertiseHost := os.Getenv("ADVERTISE_HOST")
	if advertiseHost == "" {
		advertiseHost = listenIP
	}

	// Create local (advertised) address
	addr := Address{
		IP:   advertiseHost,
		Port: *portPtr,
	}

	// Create node
	node, err := NewNode(network, addr)
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
		log.Printf("Starting as bootstrap node on %s", addr.String())
	} else {
		log.Printf("Starting node on %s, attempting to join via %s", addr.String(), *bootstrapAddrPtr)
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
	fmt.Printf("Node %s started successfully\n", addr.String())
	StartInteractiveNode(node)

	// When interactive loop ends, stop node and wait for goroutine to finish, if you have Close implemented.
	if closer, ok := interface{}(node).(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	wg.Wait()
}
