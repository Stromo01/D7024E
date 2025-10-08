package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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
func (n *Node) StoreAtK(key string, value []byte, k int) error {
	// Always store locally first
	n.StoreObject(key, value)

	// Find K closest nodes
	closest := n.routing.getKClosest(key, k)

	var wg sync.WaitGroup
	errors := make(chan error, len(closest))

	for _, contact := range closest {
		// Skip ourselves (compare IDs, not addresses)
		if bytes.Equal(contact.ID, n.id[:]) {
			continue
		}

		wg.Add(1)
		go func(c Triple) {
			defer wg.Done()
			payload := []byte(key + ":" + string(value))
			if err := n.Send(c.Addr, "store", payload); err != nil {
				errors <- fmt.Errorf("failed to store at %s: %v", c.Addr.String(), err)
			}
		}(contact)
	}

	wg.Wait()
	close(errors)

	// Collect any errors (optional - you might want to tolerate some failures)
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("storage failures: %v", errs)
	}

	return nil
}

// StoreObject stores a value by key (hash)
func (n *Node) StoreObject(key string, value []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.store[key] = value
}

// FindObject retrieves a value by key (hash)
func (n *Node) FindObjectLocally(key string) ([]byte, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	val, ok := n.store[key]
	return val, ok
}

func (n *Node) FindObject(key string) ([]byte, string, bool) {
	// First check locally
	if value, found := n.FindObjectLocally(key); found {
		return value, n.addr.String(), true
	}

	// Use iterative find value for network lookup
	if value, found := n.iterativeFindValue(key); found {
		return value, "network", true
	}

	return nil, "", false
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
		parts := strings.SplitN(string(msg.Payload), ":", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := []byte(parts[1])

			node.StoreObject(key, value)
			fmt.Printf("Node %s stored object with key %s from %s\n",
				node.Address().String(), key, msg.From.String())

			// Add the sender to routing table
			if len(msg.FromContact.ID) > 0 {
				node.routing.addContact(msg.FromContact)
			}
		}
		return nil
	})

	// Register PING handler to add sender to contacts and reply with PONG
	node.Handle(MsgPing, func(msg Message) error {
		fmt.Printf("Node %s received PING from %s (ID: %x)\n",
			node.Address().String(),
			msg.From.String(),
			msg.FromContact.ID)

		// Use the address from FromContact, not msg.From
		contactToAdd := Triple{
			ID:   msg.FromContact.ID,
			Addr: msg.FromContact.Addr, // Use this instead of msg.From
			Port: msg.FromContact.Port,
		}

		node.routing.addContact(contactToAdd)
		return node.Send(msg.FromContact.Addr, MsgPong, []byte("pong"))
	})

	// Update your PONG handler similarly
	node.Handle(MsgPong, func(msg Message) error {
		fmt.Printf("Node %s received PONG from %s (ID: %x)\n",
			node.Address().String(),
			msg.From.String(),
			msg.FromContact.ID)

		contactToAdd := Triple{
			ID:   msg.FromContact.ID,
			Addr: msg.FromContact.Addr,
			Port: msg.FromContact.Port,
		}

		node.routing.addContact(contactToAdd)
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
		key := string(msg.Payload)

		// Add the sender to routing table
		if len(msg.FromContact.ID) > 0 {
			node.routing.addContact(msg.FromContact)
		}

		// Check if we have the value
		if val, ok := node.FindObjectLocally(key); ok {
			return node.Send(msg.From, "find_value_response", []byte("VALUE:"+string(val)))
		} else {
			// Return closest nodes
			closest := node.routing.getKClosest(key, K)
			respPayload := tripleSerialize(closest)
			return node.Send(msg.From, "find_value_response", []byte(respPayload))
		}
	})
	return node, nil
}

func (n *Node) sortByDistance(key string, nodes []Triple) []Triple {
	keyBytes := []byte(key)

	sort.Slice(nodes, func(i, j int) bool {
		distI := xorDistance(keyBytes, nodes[i].ID)
		distJ := xorDistance(keyBytes, nodes[j].ID)
		return distI.Cmp(distJ) < 0
	})

	return nodes
}

func (n *Node) GetAllContacts() []Triple {
	var contacts []Triple
	for _, bucket := range n.routing.buckets {
		if bucket != nil {
			// Use GetAllContacts() method from bucket
			contacts = append(contacts, bucket.GetAllContacts()...)
		}
	}
	return contacts
}

func tripleDeserialize(s string) ([]Triple, error) {
	var triples []Triple
	addrStrs := strings.Split(s, ",")
	for _, s := range addrStrs {
		if s != "" {
			// Properly parse address string "IP:Port:ID"
			parts := strings.Split(s, ":")
			if len(parts) == 3 {
				port, err := strconv.Atoi(parts[1])
				if err != nil {
					continue
				}
				// Parse hex ID
				idBytes := make([]byte, len(parts[2])/2)
				for i := 0; i < len(parts[2]); i += 2 {
					b, err := strconv.ParseUint(parts[2][i:i+2], 16, 8)
					if err != nil {
						break
					}
					idBytes[i/2] = byte(b)
				}

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

func (n *Node) iterativeFindValue(key string) ([]byte, bool) {
	// Start with K closest nodes from routing table
	shortlist := n.routing.getKClosest(key, K)
	queried := make(map[string]bool)

	for len(shortlist) > 0 {
		// Take up to Alpha nodes to query in parallel
		toQuery := make([]Triple, 0, Alpha)
		remaining := make([]Triple, 0)

		for _, contact := range shortlist {
			contactKey := contact.Addr.String()
			if !queried[contactKey] && len(toQuery) < Alpha {
				toQuery = append(toQuery, contact)
				queried[contactKey] = true
			} else {
				remaining = append(remaining, contact)
			}
		}

		if len(toQuery) == 0 {
			break
		}

		// Query in parallel
		resultChan := make(chan findValueResult, len(toQuery))
		var wg sync.WaitGroup

		for _, contact := range toQuery {
			wg.Add(1)
			go func(c Triple) {
				defer wg.Done()
				result := n.queryNodeForValue(c, key)
				resultChan <- result
			}(contact)
		}

		wg.Wait()
		close(resultChan)

		// Process results
		var newNodes []Triple
		for result := range resultChan {
			if result.found {
				return result.value, true
			}
			newNodes = append(newNodes, result.nodes...)
		}

		// Update shortlist with new nodes
		shortlist = remaining
		for _, node := range newNodes {
			if !queried[node.Addr.String()] {
				shortlist = append(shortlist, node)
			}
		}

		// Sort by distance and keep only closest
		shortlist = n.sortByDistance(key, shortlist)
		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}

	return nil, false
}

type findValueResult struct {
	found bool
	value []byte
	nodes []Triple
}

func (n *Node) queryNodeForNodes(contact Triple, key string) []Triple {
	// Set up temporary handler for the response
	responseChan := make(chan []Triple, 1)

	// Store original handler
	originalHandler := n.handlers["find_node_response"]

	// Set temporary handler
	n.Handle("find_node_response", func(msg Message) error {
		payload := string(msg.Payload)
		nodes, err := tripleDeserialize(payload)
		if err != nil {
			responseChan <- []Triple{}
		} else {
			responseChan <- nodes
		}
		return nil
	})

	// Send the query
	err := n.Send(contact.Addr, "find_node", []byte(key))
	if err != nil {
		// Restore original handler
		n.Handle("find_node_response", originalHandler)
		return []Triple{}
	}

	// Wait for response with timeout
	select {
	case result := <-responseChan:
		// Restore original handler
		n.Handle("find_node_response", originalHandler)
		return result
	case <-time.After(5 * time.Second):
		// Timeout
		n.Handle("find_node_response", originalHandler)
		return []Triple{}
	}
}

func (n *Node) queryNodeForValue(contact Triple, key string) findValueResult {
	// Set up temporary handler for the response
	responseChan := make(chan findValueResult, 1)

	// Store original handler
	originalHandler := n.handlers["find_value_response"]

	// Set temporary handler
	n.Handle("find_value_response", func(msg Message) error {
		payload := string(msg.Payload)

		// Check if this is the value we're looking for
		if strings.HasPrefix(payload, "VALUE:") {
			value := []byte(strings.TrimPrefix(payload, "VALUE:"))
			responseChan <- findValueResult{found: true, value: value}
		} else {
			// It's a list of nodes
			nodes, _ := tripleDeserialize(payload)
			responseChan <- findValueResult{found: false, nodes: nodes}
		}
		return nil
	})

	// Send the query
	err := n.Send(contact.Addr, "find_value", []byte(key))
	if err != nil {
		// Restore original handler
		n.Handle("find_value_response", originalHandler)
		return findValueResult{found: false}
	}

	// Wait for response with timeout
	select {
	case result := <-responseChan:
		// Restore original handler
		n.Handle("find_value_response", originalHandler)
		return result
	case <-time.After(5 * time.Second):
		// Timeout
		n.Handle("find_value_response", originalHandler)
		return findValueResult{found: false}
	}
}

func tripleSerialize(triples []Triple) string {
	var respPayload string
	for i, contact := range triples {
		if i > 0 {
			respPayload += ","
		}
		respPayload += contact.Addr.String()
		respPayload += fmt.Sprintf(":%d", contact.Port)
		respPayload += fmt.Sprintf(":%x", contact.ID) // Changed from %d to %x
		//respPayload format = "addr1:port1:id1,addr2:port2:id2,..."
	}
	return respPayload
}

func (n *Node) nodeLookup(key string) []Triple {
	shortlist := n.routing.getKClosest(key, K)
	queried := make(map[string]bool)

	for {
		// Select up to Alpha unqueried nodes
		toQuery := make([]Triple, 0, Alpha)
		for _, contact := range shortlist {
			if !queried[contact.Addr.String()] && len(toQuery) < Alpha {
				toQuery = append(toQuery, contact)
				queried[contact.Addr.String()] = true
			}
		}

		if len(toQuery) == 0 {
			break
		}

		// Query in parallel
		var wg sync.WaitGroup
		nodesChan := make(chan []Triple, len(toQuery))

		for _, contact := range toQuery {
			wg.Add(1)
			go func(c Triple) {
				defer wg.Done()
				nodes := n.queryNodeForNodes(c, key)
				nodesChan <- nodes
			}(contact)
		}

		wg.Wait()
		close(nodesChan)

		// Collect new nodes
		var newNodes []Triple
		for nodes := range nodesChan {
			newNodes = append(newNodes, nodes...)
		}

		// Add new nodes to shortlist
		for _, node := range newNodes {
			if !queried[node.Addr.String()] {
				shortlist = append(shortlist, node)
			}
		}

		// Sort and trim
		shortlist = n.sortByDistance(key, shortlist)
		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}

	return shortlist
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
	return nodeDistance[:Alpha]
}

func (n *Node) iterativeStore(key string, value []byte) {
	var nodes []Triple = n.iterativeFindNode(key)
	for _, node := range nodes {
		n.Send(node.Addr, "store", value)
	}
}

func (n *Node) iterativeFindNode(key string) []Triple {
	var nodes []Triple = n.nodeLookup(key)
	for _, node := range nodes {
		n.Send(node.Addr, "find_node", []byte(key))
	}
	return nodes
}

// JoinNetwork: send PING to known node and add to contacts
func (n *Node) JoinNetwork(bootstrapNode Triple) error {
	fmt.Printf("Attempting to join network via %s\n", bootstrapNode.Addr.String())

	// Send PING to bootstrap node
	err := n.Send(bootstrapNode.Addr, MsgPing, []byte("ping"))
	if err != nil {
		return fmt.Errorf("failed to ping bootstrap node %s: %v", bootstrapNode.Addr.String(), err)
	}

	// Wait a moment for the PONG response
	time.Sleep(100 * time.Millisecond)

	// Send find_node for our own ID to populate routing table
	err = n.Send(bootstrapNode.Addr, "find_node", []byte(fmt.Sprintf("%x", n.id)))
	if err != nil {
		return fmt.Errorf("failed to send find_node to %s: %v", bootstrapNode.Addr.String(), err)
	}

	// Wait for responses
	time.Sleep(100 * time.Millisecond)

	// Send find_node for a random ID to discover more nodes
	randomID := make([]byte, 20)
	for i := range randomID {
		randomID[i] = byte(i * 13) // Simple pattern
	}
	err = n.Send(bootstrapNode.Addr, "find_node", []byte(fmt.Sprintf("%x", randomID)))
	if err != nil {
		return fmt.Errorf("failed to send second find_node to %s: %v", bootstrapNode.Addr.String(), err)
	}

	return nil
}

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
	// Format payload as "msgType:data"
	var payload []byte
	if msgType != "" {
		payload = append([]byte(msgType+":"), data...)
	} else {
		payload = data
	}

	// Create the message with proper FromContact
	msg := Message{
		From:        n.addr,
		FromContact: Triple{ID: n.id[:], Addr: n.addr, Port: n.addr.Port},
		To:          to,
		Payload:     payload,
		network:     n.network,
	}

	// Use the listening connection for sending (maintains source port)
	return n.connection.Send(msg)
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
