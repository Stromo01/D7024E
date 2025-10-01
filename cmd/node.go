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
)

// XOR distance between two keys (as hex strings)
// Removed duplicate definition of xorDistance

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

// XOR distance between two keys (as hex strings)
func xorDistance(a, b []byte) *big.Int {
	aInt := new(big.Int).SetBytes(a)
	bInt := new(big.Int).SetBytes(b)
	return new(big.Int).Xor(aInt, bInt)
}

// StoreAtK stores an object at the K closest nodes (including self if applicable)
func (n *Node) StoreAtK(key string, value []byte, k int) {
	closest := n.routing.getKClosest(key, K)
	for _, contact := range closest {
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
		closest := node.routing.getKClosest(key, K)
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
			closest := node.routing.getKClosest(key, K)
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
			// Properly parse address string "IP:Port"
			parts := strings.Split(s, ":")
			if len(parts) == 3 {
				port, err := strconv.Atoi(parts[1])
				if err != nil {
					continue
				}
				idBytes := []byte(parts[2])
				triple := Triple{
					ID:   idBytes,
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
		respPayload += contact.Addr.String()
		respPayload += fmt.Sprintf(":%d", contact.Port)
		respPayload += fmt.Sprintf(":%d", contact.ID)
		//respPayload format = "addr1:port1:id1,addr2:port2:id2,..."
	}
	return respPayload
}

func (n *Node) nodeLookup(key string) []Triple {
	search := n.routing.getKClosest(key, K)
	closestNode := search[0]
	shortlist := search[:Alpha]
	var searched []Triple
	var wg sync.WaitGroup
	responses := make(chan []Triple, len(shortlist))
	expectedResponses := 0
	var results []Triple
	for {
		n.searchShortlist(key, shortlist, responses, &wg, expectedResponses, searched) // Searches provided shortlist asynchronously
		for i := 0; i < expectedResponses; i++ {
			triples := <-responses
			shortlist = append(shortlist, triples...)
		}
		shortlist = sortAndTrim(key, shortlist)
		if bytes.Equal(closestNode.ID, shortlist[0].ID) {
			break
		} else {
			closestNode = shortlist[0]
		}
	}
	close(responses)
	return results
}

func (n *Node) searchShortlist(key string, shortlist []Triple, responses chan []Triple, wg *sync.WaitGroup, expectedResponses int, searched []Triple) {
	for _, contact := range shortlist {
		if contact.Addr == n.addr {
			continue
		}
		for _, s := range searched {
			if s.Addr == contact.Addr {
				continue
			}
		}
		expectedResponses++
		searched = append(searched, contact)
		wg.Add(1)
		go func(c Triple) {
			defer wg.Done()
			originalHandler := n.handlers["find_node_response"]
			n.Handle("find_node_response", func(msg Message) error {
				triples, err := tripleDeserialize(string(msg.Payload))
				shortlist = append(shortlist, triples...)
				if err == nil {
					responses <- triples // <-- send into the channel here
				}
				return nil
			})
			// Send find_node RPC and save it in result
			err := n.Send(c.Addr, "find_node", []byte(key))
			if err != nil {
				log.Printf("Failed to send find_node to %s: %v", c.Addr.String(), err)
				responses <- nil
				return
			}
			defer n.Handle("find_node_response", originalHandler)
		}(contact)
	}
	wg.Wait()

}

func sortAndTrim(key string, nodes []Triple) []Triple {
	var nodeDistance []Triple
	for _, node := range nodes {
		distance := xorDistance([]byte(key), node.ID)
		for i, nd := range nodeDistance {
			if distance.Cmp(xorDistance([]byte(key), nd.ID)) == -1 {
				nodeDistance = append(nodeDistance[:i], append([]Triple{node}, nodeDistance[i:]...)...)
				break
			}
		}
	}
	return nodeDistance[:K]
}

/*
// FindNode, used for finding bootstrap?
func (n *Node) FindNode(addr Address) *Triple {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, c := range n.routing {
		if c.Addr == addr {
			return &c
		}
	}
	return nil
}*/

// JoinNetwork: send PING to known node and add to contacts
func (n *Node) JoinNetwork(boostrapNode Triple) error {
	err := SendPing(n.network, n.addr, boostrapNode.Addr)
	if err != nil {
		return fmt.Errorf("failed to join network via %s: %v", boostrapNode.Addr.String(), err)
	}
	err = n.Send(boostrapNode.Addr, "find_node", []byte(fmt.Sprintf("%x", n.id)))
	if err != nil {
		return fmt.Errorf("failed to send find_node to %s: %v", boostrapNode.Addr.String(), err)
	}

	// Contact will be added when PONG is received
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
