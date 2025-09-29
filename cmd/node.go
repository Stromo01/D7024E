package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"sort"
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

	// Response tracking for async RPC operations
	pendingRPCs map[[20]byte]chan interface{} // RPC ID -> response channel
	rpcMu       sync.RWMutex
}

type Triple struct {
	ID   []byte
	Addr Address
	Port int
}

const K = 20    // Kademlia bucket size and number of closest nodes to return (as per spec)
const Alpha = 3 // Concurrency in lookups
const B = 160   // Key length in bits

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

// Public RPC Methods - These implement the core Kademlia operations

// RPCStore sends a STORE RPC to a target node
func (n *Node) RPCStore(targetAddr Address, key string, value []byte) error {
	req := StoreRequest{
		Key:   key,
		Value: value,
	}

	_, err := SendRPC(n.network, n.addr, targetAddr, MsgStore, req)
	return err
}

// RPCFindNode sends a FIND_NODE RPC to a target node and returns closest contacts
func (n *Node) RPCFindNode(targetAddr Address, searchKey []byte) ([]Triple, error) {
	req := FindNodeRequest{
		Target: searchKey,
	}

	_, err := SendRPC(n.network, n.addr, targetAddr, MsgFindNode, req)
	if err != nil {
		return nil, err
	}

	// TODO: In a full implementation, we'd wait for the response with matching RPC ID
	// For now, we'll return what we have in our routing table
	return n.routing.getKClosest(string(searchKey)), nil
}

// RPCFindValue sends a FIND_VALUE RPC to a target node
func (n *Node) RPCFindValue(targetAddr Address, key string) ([]byte, []Triple, bool, error) {
	req := FindValueRequest{
		Key: key,
	}

	_, err := SendRPC(n.network, n.addr, targetAddr, MsgFindValue, req)
	if err != nil {
		return nil, nil, false, err
	}

	// TODO: In a full implementation, we'd wait for the response with matching RPC ID
	// For now, check if we have it locally
	if value, found := n.FindObject(key); found {
		return value, nil, true, nil
	}

	// Return closest contacts if not found
	contacts := n.routing.getKClosest(key)
	return nil, contacts, false, nil
}

// RPCPing sends a PING RPC to a target node
func (n *Node) RPCPing(targetAddr Address) error {
	_, err := SendRPC(n.network, n.addr, targetAddr, MsgPing, nil)
	return err
}

// Iterative Operations - The core of Kademlia DHT

// IterativeFindNode performs the iterative node lookup operation
func (n *Node) IterativeFindNode(targetKey []byte) []Triple {
	// Start with closest contacts from local routing table
	shortlist := n.routing.getKClosest(string(targetKey))
	if len(shortlist) == 0 {
		return []Triple{}
	}

	// Keep track of contacted nodes to avoid duplicates
	contacted := make(map[string]bool)
	var closestNode *Triple
	var closestDistance *big.Int

	// Set initial closest
	if len(shortlist) > 0 {
		closestNode = &shortlist[0]
		closestDistance = xorDistance(targetKey, closestNode.ID)
	}

	for {
		// Select up to Alpha nodes to contact in parallel (that haven't been contacted)
		var toContact []Triple
		for _, contact := range shortlist {
			if len(toContact) >= Alpha {
				break
			}
			contactKey := contact.Addr.String()
			if !contacted[contactKey] && contact.Addr != n.addr {
				toContact = append(toContact, contact)
				contacted[contactKey] = true
			}
		}

		if len(toContact) == 0 {
			break // No more nodes to contact
		}

		// Send FIND_NODE RPCs in parallel
		type result struct {
			contacts []Triple
			err      error
		}

		results := make(chan result, len(toContact))

		for _, contact := range toContact {
			go func(addr Address) {
				req := FindNodeRequest{Target: targetKey}
				resp, err := n.sendRPCWithResponse(addr, MsgFindNode, req, 5*time.Second)

				var contacts []Triple
				if err == nil && resp != nil {
					// Parse response
					if findNodeResp, ok := resp.(map[string]interface{}); ok {
						if contactsData, exists := findNodeResp["contacts"]; exists {
							// Convert to []Triple (this is a simplified conversion)
							if contactsList, ok := contactsData.([]interface{}); ok {
								for _, c := range contactsList {
									if contactMap, ok := c.(map[string]interface{}); ok {
										if id, hasID := contactMap["ID"].(string); hasID {
											if addr, hasAddr := contactMap["Addr"].(map[string]interface{}); hasAddr {
												if ip, hasIP := addr["IP"].(string); hasIP {
													if port, hasPort := addr["Port"].(float64); hasPort {
														contact := Triple{
															ID:   []byte(id),
															Addr: Address{IP: ip, Port: int(port)},
															Port: int(port),
														}
														contacts = append(contacts, contact)
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}

				results <- result{contacts: contacts, err: err}
			}(contact.Addr)
		}

		// Collect results
		var newContacts []Triple
		foundCloser := false

		for i := 0; i < len(toContact); i++ {
			result := <-results
			if result.err == nil {
				for _, contact := range result.contacts {
					// Add to routing table
					n.routing.addContact(contact)

					// Check if this is closer than our current closest
					dist := xorDistance(targetKey, contact.ID)
					if closestDistance == nil || dist.Cmp(closestDistance) < 0 {
						closestNode = &contact
						closestDistance = dist
						foundCloser = true
					}

					newContacts = append(newContacts, contact)
				}
			}
		}

		// Update shortlist with all known contacts, sorted by distance
		allContacts := append(shortlist, newContacts...)
		shortlist = n.sortContactsByDistance(targetKey, allContacts)

		// Keep only the K closest
		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}

		// If we haven't found a closer node, query the K closest that we haven't contacted
		if !foundCloser {
			break
		}
	}

	return shortlist
}

// IterativeFindValue performs the iterative value lookup operation
func (n *Node) IterativeFindValue(key string) ([]byte, []Triple, bool) {
	targetKey := []byte(key)

	// Check locally first
	if value, found := n.FindObject(key); found {
		return value, nil, true
	}

	// Start with closest contacts from local routing table
	shortlist := n.routing.getKClosest(key)
	if len(shortlist) == 0 {
		return nil, nil, false
	}

	// Keep track of contacted nodes
	contacted := make(map[string]bool)

	for {
		// Select up to Alpha nodes to contact in parallel
		var toContact []Triple
		for _, contact := range shortlist {
			if len(toContact) >= Alpha {
				break
			}
			contactKey := contact.Addr.String()
			if !contacted[contactKey] && contact.Addr != n.addr {
				toContact = append(toContact, contact)
				contacted[contactKey] = true
			}
		}

		if len(toContact) == 0 {
			break
		}

		// Send FIND_VALUE RPCs in parallel
		type result struct {
			value    []byte
			contacts []Triple
			found    bool
			err      error
		}

		results := make(chan result, len(toContact))

		for _, contact := range toContact {
			go func(addr Address) {
				req := FindValueRequest{Key: key}
				resp, err := n.sendRPCWithResponse(addr, MsgFindValue, req, 5*time.Second)

				var value []byte
				var contacts []Triple
				found := false

				if err == nil && resp != nil {
					// Parse response properly using JSON unmarshaling
					data, marshalErr := json.Marshal(resp)
					if marshalErr == nil {
						var findValueResp FindValueResponse
						if unmarshalErr := json.Unmarshal(data, &findValueResp); unmarshalErr == nil {
							if findValueResp.Found {
								value = findValueResp.Value
								found = true
							} else {
								contacts = findValueResp.Contacts
							}
						}
					}
				}

				results <- result{value: value, contacts: contacts, found: found, err: err}
			}(contact.Addr)
		}

		// Collect results
		for i := 0; i < len(toContact); i++ {
			result := <-results
			if result.err == nil {
				if result.found {
					// Value found! Store it at the first available closest node
					if len(shortlist) > 0 {
						n.RPCStore(shortlist[0].Addr, key, result.value)
					}
					return result.value, shortlist, true
				}

				// Add new contacts
				for _, contact := range result.contacts {
					n.routing.addContact(contact)
					shortlist = append(shortlist, contact)
				}
			}
		}

		// Update shortlist
		shortlist = n.sortContactsByDistance(targetKey, shortlist)
		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}

	return nil, shortlist, false
}

// IterativeStore stores a key-value pair at the K closest nodes
func (n *Node) IterativeStore(key string, value []byte) error {
	// Find the K closest nodes to the key
	closestNodes := n.IterativeFindNode([]byte(key))

	if len(closestNodes) == 0 {
		return fmt.Errorf("no nodes found to store at")
	}

	// Store at up to K closest nodes
	storeCount := len(closestNodes)
	if storeCount > K {
		storeCount = K
	}

	errors := make(chan error, storeCount)

	// Store in parallel
	for i := 0; i < storeCount; i++ {
		go func(addr Address) {
			err := n.RPCStore(addr, key, value)
			errors <- err
		}(closestNodes[i].Addr)
	}

	// Collect results
	var lastError error
	successCount := 0
	for i := 0; i < storeCount; i++ {
		err := <-errors
		if err == nil {
			successCount++
		} else {
			lastError = err
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to store at any node: %v", lastError)
	}

	fmt.Printf("Successfully stored at %d/%d nodes\n", successCount, storeCount)
	return nil
}

// Helper function to sort contacts by distance to target
func (n *Node) sortContactsByDistance(targetKey []byte, contacts []Triple) []Triple {
	type contactDistance struct {
		contact Triple
		dist    *big.Int
	}

	var candidates []contactDistance
	seen := make(map[string]bool)

	for _, contact := range contacts {
		contactKey := contact.Addr.String()
		if !seen[contactKey] {
			seen[contactKey] = true
			dist := xorDistance(targetKey, contact.ID)
			candidates = append(candidates, contactDistance{
				contact: contact,
				dist:    dist,
			})
		}
	}

	// Sort by distance
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dist.Cmp(candidates[j].dist) < 0
	})

	// Extract contacts
	result := make([]Triple, len(candidates))
	for i, candidate := range candidates {
		result[i] = candidate.contact
	}

	return result
}

// sendRPCWithResponse sends an RPC and waits for a response with timeout
func (n *Node) sendRPCWithResponse(targetAddr Address, rpcType string, payload interface{}, timeout time.Duration) (interface{}, error) {
	// Create response channel
	responseChan := make(chan interface{}, 1)

	// Send RPC and get ID
	rpcID, err := SendRPC(n.network, n.addr, targetAddr, rpcType, payload)
	if err != nil {
		return nil, err
	}

	// Register for response
	n.rpcMu.Lock()
	n.pendingRPCs[rpcID] = responseChan
	n.rpcMu.Unlock()

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		// Cleanup on timeout
		n.rpcMu.Lock()
		delete(n.pendingRPCs, rpcID)
		close(responseChan)
		n.rpcMu.Unlock()
		return nil, fmt.Errorf("RPC timeout")
	}
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
		id:          id,
		addr:        addr,
		network:     network,
		connection:  connection,
		handlers:    make(map[string]MessageHandler),
		routing:     NewRoutingTable(Triple{ID: id[:], Addr: addr, Port: addr.Port}),
		store:       make(map[string][]byte),
		pendingRPCs: make(map[[20]byte]chan interface{}),
	}

	// Register RPC handlers
	node.setupRPCHandlers()
	return node, nil
}

// setupRPCHandlers registers all the core Kademlia RPC handlers
func (n *Node) setupRPCHandlers() {
	// Handle RPC messages by message type
	n.Handle(MsgStore, func(msg Message) error {
		return n.handleRPCMessage(msg)
	})

	n.Handle(MsgFindNode, func(msg Message) error {
		return n.handleRPCMessage(msg)
	})

	n.Handle(MsgFindValue, func(msg Message) error {
		return n.handleRPCMessage(msg)
	})

	// Handle RPC responses
	n.Handle(MsgStoreResp, func(msg Message) error {
		return n.handleRPCMessage(msg)
	})

	n.Handle(MsgFindNodeResp, func(msg Message) error {
		return n.handleRPCMessage(msg)
	})

	n.Handle(MsgFindValueResp, func(msg Message) error {
		return n.handleRPCMessage(msg)
	})

	// For backward compatibility, also handle raw message types
	n.Handle(MsgPing, func(msg Message) error {
		return n.handlePing(msg)
	})

	n.Handle(MsgPong, func(msg Message) error {
		return n.handlePong(msg)
	})
}

// handleRPCMessage parses an RPC message and routes it to the appropriate handler
func (n *Node) handleRPCMessage(msg Message) error {
	var rpc RPC
	err := json.Unmarshal(msg.Payload, &rpc)
	if err != nil {
		// Try to handle as legacy message format
		return n.handleLegacyMessage(msg)
	}

	// Update routing table with sender information
	n.routing.addContact(msg.FromContact)

	switch rpc.Type {
	case MsgPing:
		return n.handleRPCPing(msg, rpc)
	case MsgStore:
		return n.handleRPCStore(msg, rpc)
	case MsgFindNode:
		return n.handleRPCFindNode(msg, rpc)
	case MsgFindValue:
		return n.handleRPCFindValue(msg, rpc)
	case MsgPongResp, MsgStoreResp, MsgFindNodeResp, MsgFindValueResp:
		// Handle responses (these might be handled by specific waiting goroutines)
		return n.handleRPCResponse(msg, rpc)
	default:
		return fmt.Errorf("unknown RPC type: %s", rpc.Type)
	}
}

// handleLegacyMessage handles old-style string messages for backward compatibility
func (n *Node) handleLegacyMessage(msg Message) error {
	payload := string(msg.Payload)

	if payload == MsgPing {
		return n.handlePing(msg)
	} else if payload == MsgPong || strings.HasPrefix(payload, MsgPong+":") {
		return n.handlePong(msg)
	}

	return fmt.Errorf("unknown legacy message: %s", payload)
}

// Core RPC Handlers

// handleRPCPing handles PING RPC requests
func (n *Node) handleRPCPing(msg Message, rpc RPC) error {
	fmt.Printf("Node %s received PING RPC from %s\n", n.Address().String(), msg.From.String())

	// Send PONG response
	response := struct{}{}
	return msg.ReplyRPC(rpc.ID, MsgPongResp, response)
}

// handleRPCStore handles STORE RPC requests
func (n *Node) handleRPCStore(msg Message, rpc RPC) error {
	var req StoreRequest
	data, err := json.Marshal(rpc.Payload)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &req)
	if err != nil {
		response := StoreResponse{Success: false, Error: "invalid store request format"}
		return msg.ReplyRPC(rpc.ID, MsgStoreResp, response)
	}

	// Store the key-value pair
	n.StoreObject(req.Key, req.Value)
	fmt.Printf("Node %s stored object with key %s from %s\n", n.Address().String(), req.Key, msg.From.String())

	// Send success response
	response := StoreResponse{Success: true}
	return msg.ReplyRPC(rpc.ID, MsgStoreResp, response)
}

// handleRPCFindNode handles FIND_NODE RPC requests
func (n *Node) handleRPCFindNode(msg Message, rpc RPC) error {
	var req FindNodeRequest
	data, err := json.Marshal(rpc.Payload)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &req)
	if err != nil {
		return fmt.Errorf("invalid find_node request format: %v", err)
	}

	// Find closest contacts to the target
	target := string(req.Target)
	closest := n.routing.getKClosest(target)

	// Limit to K contacts as per spec
	if len(closest) > K {
		closest = closest[:K]
	}

	response := FindNodeResponse{Contacts: closest}
	return msg.ReplyRPC(rpc.ID, MsgFindNodeResp, response)
}

// handleRPCFindValue handles FIND_VALUE RPC requests
func (n *Node) handleRPCFindValue(msg Message, rpc RPC) error {
	var req FindValueRequest
	data, err := json.Marshal(rpc.Payload)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &req)
	if err != nil {
		return fmt.Errorf("invalid find_value request format: %v", err)
	}

	// Check if we have the value
	if value, found := n.FindObject(req.Key); found {
		response := FindValueResponse{Found: true, Value: value}
		return msg.ReplyRPC(rpc.ID, MsgFindValueResp, response)
	}

	// If not found, return closest contacts (same as FIND_NODE)
	closest := n.routing.getKClosest(req.Key)
	if len(closest) > K {
		closest = closest[:K]
	}

	response := FindValueResponse{Found: false, Contacts: closest}
	return msg.ReplyRPC(rpc.ID, MsgFindValueResp, response)
}

// handleRPCResponse handles RPC responses (for matching with pending requests)
func (n *Node) handleRPCResponse(msg Message, rpc RPC) error {
	// Find the pending RPC and send the response
	n.rpcMu.RLock()
	responseChan, exists := n.pendingRPCs[rpc.ID]
	n.rpcMu.RUnlock()

	if exists {
		// Send the response to the waiting goroutine
		select {
		case responseChan <- rpc.Payload:
			// Response sent successfully
		default:
			// Response channel is full or closed, ignore
		}

		// Clean up the pending RPC
		n.rpcMu.Lock()
		delete(n.pendingRPCs, rpc.ID)
		close(responseChan)
		n.rpcMu.Unlock()
	}

	return nil
}

// Legacy handlers for backward compatibility
func (n *Node) handlePing(msg Message) error {
	n.routing.addContact(msg.FromContact)
	fmt.Printf("Node %s received PING from %s\n", n.Address().String(), msg.From.String())
	return n.Send(msg.From, MsgPong, []byte("pong"))
}

func (n *Node) handlePong(msg Message) error {
	n.routing.addContact(msg.FromContact)
	fmt.Printf("Node %s received PONG from %s\n", n.Address().String(), msg.From.String())
	return nil
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
