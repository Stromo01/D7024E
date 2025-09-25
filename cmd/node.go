package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

// XOR distance between two keys (as byte arrays)
func xorDistance(a, b []byte) *big.Int {
	// Create a new byte array for the result
	result := make([]byte, IDLength)

	// Ensure we don't go out of bounds
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	if minLen > IDLength {
		minLen = IDLength
	}

	// Perform XOR operation on each byte
	for i := 0; i < minLen; i++ {
		result[i] = a[i] ^ b[i]
	}

	// Convert to big.Int and return
	return new(big.Int).SetBytes(result)
}

type Node struct {
	id         [20]byte // 160 bits
	addr       Address
	network    Network
	connection Connection
	handlers   map[string]MessageHandler
	store      map[string][]byte // Object store: key (hash) -> value
	routing    *RoutingTable
	mu         sync.RWMutex
	closed     bool
	closeMu    sync.RWMutex
}

type Triple struct {
	ID   []byte
	Addr Address
	Port int
}

const K = 8     // Kademlia bucket size and number of closest nodes to return
const Alpha = 3 // Concurrency in lookups

// StoreAtK stores an object at the K closest nodes (including self if applicable)
func (n *Node) StoreAtK(key string, value []byte, k int) {
	// Get the closest nodes but limit to k nodes
	allNodes := n.routing.getKClosest(key)
	var closestNodes []Triple

	if len(allNodes) > k {
		closestNodes = allNodes[:k]
	} else {
		closestNodes = allNodes
	}

	// Store at each of the k closest nodes
	for _, contact := range closestNodes {
		if contact.Addr == n.addr {
			n.StoreObject(key, value)
		} else {
			payload := []byte(key + ":" + string(value))
			n.Send(contact.Addr, "store", payload)
		}
	}
}

// StoreObject stores a value by key (hash)
func (n *Node) StoreObject(key string, value []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.store[key] = value
}

// FindObject retrieves a value by key (hash)
func (n *Node) FindObject(key string) ([]byte, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	val, ok := n.store[key]
	return val, ok
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg Message) error

// NewNode creates a new node that can both send and receive messages
func NewNode(network Network, addr Address) (*Node, error) {
	connection, err := network.Listen(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}
	var id [20]byte
	_, err = rand.Read(id[:])
	if err != nil {
		return nil, fmt.Errorf("failed to generate node ID: %v", err)
	}
	node := &Node{
		id:         id,
		addr:       addr,
		network:    network,
		connection: connection,
		handlers:   make(map[string]MessageHandler),
		routing:    NewRoutingTable(Triple{ID: id[:], Addr: addr, Port: addr.Port}),
		store:      make(map[string][]byte),
	}

	// Register STORE handler to accept and store objects
	node.Handle("store", func(msg Message) error {
		// Expect payload as "key:value"
		payload := string(msg.Payload)
		sep := -1
		for i, c := range payload {
			if c == ':' {
				sep = i
				break
			}
		}
		if sep == -1 {
			return fmt.Errorf("invalid store payload, missing ':' separator")
		}
		key := payload[:sep]
		value := []byte(payload[sep+1:])
		node.StoreObject(key, value)
		fmt.Printf("Node %s stored object with key %s from %s\n", node.Address().String(), key, msg.From.String())
		return nil
	})

	// Register PING handler to add sender to contacts and reply with PONG
	node.Handle(MsgPing, func(msg Message) error {
		node.routing.addContact(msg.FromContact)
		fmt.Printf("Node %s received PING from %s\n", node.Address().String(), msg.From.String())
		return node.Send(msg.From, MsgPong, []byte("pong"))
	})

	// Register PONG handler (for logging and adding contact)
	node.Handle(MsgPong, func(msg Message) error {
		node.routing.addContact(msg.FromContact)
		fmt.Printf("Node %s received PONG from %s\n", node.Address().String(), msg.From.String())
		return nil
	})

	node.Handle("find_node", func(msg Message) error {
		// Expect payload as "key"
		key := string(msg.Payload)
		closest := node.routing.getKClosest(key)
		var respPayload = tripleSerialize(closest)
		return node.Send(msg.From, "find_node_response", []byte(respPayload))
	})

	node.Handle("find_node_response", func(msg Message) error {
		// Expect payload as "addr1:port1:id1,addr2:port2:id2,..."
		payload := string(msg.Payload)
		if payload != "" {
			triples, err := tripleDeserialize(payload)
			if err != nil {
				return fmt.Errorf("invalid find_node_response payload: %v", err)
			}
			for _, t := range triples {
				node.routing.addContact(t)
			}
		}
		return nil
	})

	node.Handle("find_value", func(msg Message) error {
		// Expect payload as "key"
		key := string(msg.Payload)
		if val, ok := node.FindObject(key); ok { //TODO: Switch to routing table
			return node.Send(msg.From, "find_value_response", val)
		} else {
			// Expect payload as "key"
			key := string(msg.Payload)
			closest := node.routing.getKClosest(key)
			var respPayload = tripleSerialize(closest)
			return node.Send(msg.From, "find_node_response", []byte(respPayload))
		}
	})
	return node, nil
}

func tripleDeserialize(s string) ([]Triple, error) {
	var triples []Triple
	addrStrs := strings.Split(s, ",")
	for _, s := range addrStrs {
		if s != "" {
			// Properly parse address string "IP:Port:ID"
			parts := strings.Split(s, ":")
			if len(parts) >= 3 {
				port, err := strconv.Atoi(parts[1])
				if err != nil {
					continue
				}

				// The rest of the string after IP:port: is the ID
				idStr := strings.Join(parts[2:], ":")
				triple := Triple{
					ID:   []byte(idStr),
					Addr: Address{IP: parts[0], Port: port},
					Port: port,
				}
				triples = append(triples, triple)
			}
		}
	}
	return triples, nil
}

func tripleSerialize(triples []Triple) string {
	var respPayload string
	for i, contact := range triples {
		if i > 0 {
			respPayload += ","
		}
		// Format: IP:Port:ID
		respPayload += contact.Addr.IP
		respPayload += fmt.Sprintf(":%d", contact.Port)
		respPayload += fmt.Sprintf(":%s", string(contact.ID))
	}
	return respPayload
}

func (n *Node) nodeLookup(key string) []Triple {
	search := n.routing.getKClosest(key)

	// If we have fewer results than Alpha, use all of them
	alphaCount := Alpha
	if len(search) < Alpha {
		alphaCount = len(search)
	}

	shortlist := search[:alphaCount]
	var searched []Triple
	var wg sync.WaitGroup
	responses := make(chan []Triple, Alpha) // Limit channel size to Alpha
	expectedResponses := 0

	// Store the original handler to restore it later
	originalHandler := n.handlers["find_node_response"]

	// Set up the temporary handler for responses
	n.Handle("find_node_response", func(msg Message) error {
		triples, err := tripleDeserialize(string(msg.Payload))
		if err == nil {
			responses <- triples
		} else {
			responses <- nil // Send nil if there's an error
		}
		return nil
	})

	for _, contact := range shortlist {
		if contact.Addr == n.addr {
			continue // Skip self
		}

		// Skip nodes we've already searched
		alreadySearched := false
		for _, s := range searched {
			if s.Addr == contact.Addr {
				alreadySearched = true
				break
			}
		}
		if alreadySearched {
			continue
		}

		expectedResponses++
		searched = append(searched, contact)
		wg.Add(1)

		go func(c Triple) {
			defer wg.Done()
			// Send find_node RPC
			err := n.Send(c.Addr, "find_node", []byte(key))
			if err != nil {
				log.Printf("Failed to send find_node to %s: %v", c.Addr.String(), err)
			}
		}(contact)
	}

	// Create a timeout channel
	timeout := time.After(2 * time.Second)

	// Wait for all goroutines to finish sending requests
	wg.Wait()

	// Collect responses with timeout
	var allTriples []Triple
	for i := 0; i < expectedResponses; i++ {
		select {
		case triples := <-responses:
			if triples != nil {
				allTriples = append(allTriples, triples...)
			}
		case <-timeout:
			// Stop waiting for more responses if we hit the timeout
			log.Printf("Timeout waiting for nodeLookup responses")
			i = expectedResponses // Break out of the loop
		}
	}

	// Restore the original handler
	n.Handle("find_node_response", originalHandler)

	// Add any new contacts to shortlist
	for _, triple := range allTriples {
		isNew := true
		for _, existing := range shortlist {
			if bytes.Equal(triple.ID, existing.ID) {
				isNew = false
				break
			}
		}
		if isNew {
			shortlist = append(shortlist, triple)
		}
	}

	// Return all the nodes we found
	return shortlist
}

// JoinNetwork: send PING to known node and add to contacts
func (n *Node) JoinNetwork(boostrapNode Triple) error {
	// First add the bootstrap node to our routing table
	n.routing.addContact(boostrapNode)

	// Send a message with our information included
	err := n.Send(boostrapNode.Addr, MsgPing, []byte("joining"))
	if err != nil {
		return fmt.Errorf("failed to send ping to %s: %v", boostrapNode.Addr.String(), err)
	}

	// Set the proper FromContact for the find_node message
	err = n.Send(boostrapNode.Addr, "find_node", []byte(fmt.Sprintf("%x", n.id)))
	if err != nil {
		return fmt.Errorf("failed to send find_node to %s: %v", boostrapNode.Addr.String(), err)
	}

	// The contact will be added when a PONG is received
	return nil
}

/*
func (n *Node) pingAllKnown(nodes []Triple) {
	for _, node := range nodes {
		err := SendPing(n.network, n.addr, node.Addr)
		if err != nil {
			log.Printf("Failed to ping known node %s: %v", node.Addr.String(), err)
		}
	}
}*/

// Handle registers a message handler for a specific message type
func (n *Node) Handle(msgType string, handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[msgType] = handler
}

// Start begins listening for incoming messages
func (n *Node) Start() {
	go func() {
		for {
			n.closeMu.RLock()
			if n.closed {
				n.closeMu.RUnlock()
				return
			}
			n.closeMu.RUnlock()

			msg, err := n.connection.Recv()
			if err != nil {
				n.closeMu.RLock()
				if !n.closed {
					log.Printf("Node %s failed to receive message: %v", n.addr.String(), err)
				}
				n.closeMu.RUnlock()
				return
			}

			// Extract message type from payload (first part before ':')
			msgType := "default"
			payload := string(msg.Payload)
			if len(payload) > 0 {
				for i, char := range payload {
					if char == ':' {
						msgType = payload[:i]
						// Pass only the payload after the first colon to the handler
						msg.Payload = []byte(payload[i+1:])
						break
					}
				}
			}

			n.mu.RLock()
			handler, exists := n.handlers[msgType]
			if !exists {
				handler, exists = n.handlers["default"]
			}
			n.mu.RUnlock()

			if exists && handler != nil {
				if err := handler(msg); err != nil {
					log.Printf("Handler error: %v", err)
				}
			}
		}
	}()
}

// Send sends a message to the target address
func (n *Node) Send(to Address, msgType string, data []byte) error {
	connection, err := n.network.Dial(to)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", to.String(), err)
	}
	defer connection.Close()

	// Format payload as "msgType:data"
	var payload []byte
	if msgType != "" {
		payload = append([]byte(msgType+":"), data...)
	} else {
		payload = data
	}

	msg := Message{
		From:    n.addr,
		To:      to,
		Payload: payload,
		// Include sender's contact information for routing table updates
		FromContact: Triple{
			ID:   n.id[:],
			Addr: n.addr,
			Port: n.addr.Port,
		},
	}

	return connection.Send(msg)
}

// SendString is a convenience method for sending string messages
func (n *Node) SendString(to Address, msgType, data string) error {
	return n.Send(to, msgType, []byte(data))
}

// Close shuts down the node
func (n *Node) Close() error {
	n.closeMu.Lock()
	n.closed = true
	n.closeMu.Unlock()
	return n.connection.Close()
}

// Address returns the node's address
func (n *Node) Address() Address {
	return n.addr
}
