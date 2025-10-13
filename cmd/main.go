package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/eislab-cps/go-template/internal/cli"
	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/internal/node"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

func main() {
	fmt.Print("Starting Kademlia node...\n")
	node := startKademliaNode()
	fmt.Print("Starting Kademlia CLI...\n")

	// All nodes run CLI - they stay alive until user types "exit"
	cli.StartInteractiveCLI(node)

	fmt.Println("Node shutting down...")
}

func startKademliaNode() *Node {
	// Command line flags
	portPtr := flag.Int("port", 8000, "Port to listen on")
	bootstrapAddrPtr := flag.String("bootstrap-addr", "127.0.0.1:8000", "Bootstrap node address (ip:port)")
	isBootstrapPtr := flag.Bool("bootstrap", false, "Run as bootstrap node")

	flag.Parse()

	// Override with environment variables
	if v := os.Getenv("NODE_PORT"); v != "" {
		if p, _ := strconv.Atoi(v); p > 0 {
			*portPtr = p
		}
	}
	if os.Getenv("IS_BOOTSTRAP") == "true" {
		*isBootstrapPtr = true
	}
	if v := os.Getenv("BOOTSTRAP_ADDR"); v != "" {
		*bootstrapAddrPtr = v
	}

	log.Printf("Port: %d, IsBootstrap: %t, BootstrapAddr: %s", *portPtr, *isBootstrapPtr, *bootstrapAddrPtr)

	network := NewUDPNetwork()

	advertiseHost := os.Getenv("ADVERTISE_HOST")
	if advertiseHost == "" {
		advertiseHost = outboundIP()
	}

	advertiseAddr := Address{IP: advertiseHost, Port: *portPtr}
	log.Printf("Creating node with advertise address: %s", advertiseAddr.String())

	node, err := NewNode(network, advertiseAddr)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	// Start the node in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Node crashed with panic: %v", r)
			}
		}()

		log.Printf("Starting node listener...")
		node.Start()
	}()

	// Bootstrap logic
	if *isBootstrapPtr {
		log.Printf("Starting as bootstrap node on %s", advertiseAddr.String())
	} else {
		log.Printf("Starting node on %s, joining via %s", advertiseAddr.String(), *bootstrapAddrPtr)
		bootAddrParts := strings.Split(*bootstrapAddrPtr, ":")
		if len(bootAddrParts) == 2 {
			if bootPort, err := strconv.Atoi(bootAddrParts[1]); err == nil {
				bootAddr := Address{IP: bootAddrParts[0], Port: bootPort}
				bootTriple := Triple{Addr: bootAddr, Port: bootPort}
				if err := node.JoinNetwork(bootTriple); err != nil {
					log.Printf("Failed to join network: %v", err)
				} else {
					log.Printf("Joined network via %s", *bootstrapAddrPtr)
				}
			}
		}
	}

	fmt.Printf("Node %s started successfully\n", advertiseAddr.String())
	return node
}

func outboundIP() string {
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer c.Close()
	if ua, ok := c.LocalAddr().(*net.UDPAddr); ok {
		return ua.IP.String()
	}
	return "127.0.0.1"
}
