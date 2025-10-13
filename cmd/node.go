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

type Node struct {
	id         [20]byte
	addr       Address
	network    Network
	connection Connection
	handlers   map[string]MessageHandler
	store      map[string][]byte
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

type MessageHandler func(msg Message) error

const K = 8
const Alpha = 3

// NewNode creates a new node
func NewNode(network Network, addr Address) (*Node, error) {
	connection, err := network.Listen(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}

	var id [20]byte
	rand.Read(id[:])

	node := &Node{
		id:         id,
		addr:       addr,
		network:    network,
		connection: connection,
		handlers:   make(map[string]MessageHandler),
		routing:    NewRoutingTable(Triple{ID: id[:], Addr: addr, Port: addr.Port}),
		store:      make(map[string][]byte),
	}

	// Register handlers
	node.registerHandlers()
	return node, nil
}

// Start listening for messages
func (n *Node) Start() {
	for {
		n.closeMu.RLock()
		if n.closed {
			n.closeMu.RUnlock()
			return
		}
		n.closeMu.RUnlock()

		msg, err := n.connection.Recv()
		if err != nil {
			if !n.closed {
				log.Printf("Node %s recv error: %v", n.addr.String(), err)
			}
			return
		}

		// Handle message
		msgType := strings.SplitN(string(msg.Payload), ":", 2)[0]

		n.mu.RLock()
		handler := n.handlers[msgType]
		n.mu.RUnlock()

		if handler != nil {
			handler(msg)
		}
	}
}

// Store object locally
func (n *Node) StoreObject(key string, value []byte) {
	n.mu.Lock()
	n.store[key] = value
	n.mu.Unlock()
}

// Find object locally
func (n *Node) FindObjectLocally(key string) ([]byte, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	val, ok := n.store[key]
	return val, ok
}

// Store at K closest nodes
func (n *Node) StoreAtK(key string, value []byte, k int) error {
	n.StoreObject(key, value)
	closest := n.routing.getKClosest(key, k)

	var wg sync.WaitGroup
	for _, contact := range closest {
		if bytes.Equal(contact.ID, n.id[:]) {
			continue
		}
		wg.Add(1)
		go func(c Triple) {
			defer wg.Done()
			payload := key + ":" + string(value)
			n.Send(c.Addr, "store", []byte(payload))
		}(contact)
	}
	wg.Wait()
	return nil
}

// Find object in network
func (n *Node) FindObject(key string) ([]byte, string, bool) {
	// Check locally first
	if value, found := n.FindObjectLocally(key); found {
		return value, n.addr.String(), true
	}

	// Search network
	closest := n.nodeLookup(key)
	for _, contact := range closest {
		if bytes.Equal(contact.ID, n.id[:]) {
			continue
		}
		if value := n.queryForValue(contact, key); value != nil {
			return value, contact.Addr.String(), true
		}
	}
	return nil, "", false
}

// Iterative node lookup
func (n *Node) nodeLookup(key string) []Triple {
	shortlist := n.routing.getKClosest(key, K)
	queried := make(map[string]bool)

	for {
		toQuery := []Triple{}
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
				nodes := n.queryForNodes(c, key)
				nodesChan <- nodes
			}(contact)
		}

		wg.Wait()
		close(nodesChan)

		// Collect and sort results
		for nodes := range nodesChan {
			shortlist = append(shortlist, nodes...)
		}
		shortlist = n.sortByDistance(key, shortlist)
		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}
	return shortlist
}

// Join network via bootstrap node
func (n *Node) JoinNetwork(bootstrapNode Triple) error {
	err := n.Send(bootstrapNode.Addr, MsgPing, []byte("ping"))
	if err != nil {
		return fmt.Errorf("failed to ping bootstrap: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Find nodes
	n.Send(bootstrapNode.Addr, "find_node", []byte(fmt.Sprintf("%x", n.id)))
	return nil
}

// Send message
func (n *Node) Send(to Address, msgType string, data []byte) error {
	payload := append([]byte(msgType+":"), data...)
	msg := Message{
		From:        n.addr,
		FromContact: Triple{ID: n.id[:], Addr: n.addr, Port: n.addr.Port},
		To:          to,
		Payload:     payload,
		network:     n.network,
	}
	return n.connection.Send(msg)
}

// Register message handlers
func (n *Node) Handle(msgType string, handler MessageHandler) {
	n.mu.Lock()
	n.handlers[msgType] = handler
	n.mu.Unlock()
}

// Helper functions
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
			contacts = append(contacts, bucket.GetAllContacts()...)
		}
	}
	return contacts
}

func (n *Node) queryForNodes(contact Triple, key string) []Triple {
	responseChan := make(chan []Triple, 1)

	// Temporary handler
	originalHandler := n.handlers["find_node_response"]
	n.Handle("find_node_response", func(msg Message) error {
		nodes, _ := tripleDeserialize(string(msg.Payload))
		responseChan <- nodes
		return nil
	})

	n.Send(contact.Addr, "find_node", []byte(key))

	select {
	case result := <-responseChan:
		n.Handle("find_node_response", originalHandler)
		return result
	case <-time.After(5 * time.Second):
		n.Handle("find_node_response", originalHandler)
		return []Triple{}
	}
}

func (n *Node) queryForValue(contact Triple, key string) []byte {
	responseChan := make(chan []byte, 1)

	originalHandler := n.handlers["find_value_response"]
	n.Handle("find_value_response", func(msg Message) error {
		payload := string(msg.Payload)
		if strings.HasPrefix(payload, "VALUE:") {
			responseChan <- []byte(strings.TrimPrefix(payload, "VALUE:"))
		} else {
			responseChan <- nil
		}
		return nil
	})

	n.Send(contact.Addr, "find_value", []byte(key))

	select {
	case result := <-responseChan:
		n.Handle("find_value_response", originalHandler)
		return result
	case <-time.After(5 * time.Second):
		n.Handle("find_value_response", originalHandler)
		return nil
	}
}

// Register all message handlers
func (n *Node) registerHandlers() {
	n.Handle("store", func(msg Message) error {
		parts := strings.SplitN(string(msg.Payload), ":", 2)
		if len(parts) == 2 {
			n.StoreObject(parts[0], []byte(parts[1]))
			n.routing.addContact(msg.FromContact)
		}
		return nil
	})

	n.Handle(MsgPing, func(msg Message) error {
		n.routing.addContact(msg.FromContact)
		return n.Send(msg.FromContact.Addr, MsgPong, []byte("pong"))
	})

	n.Handle(MsgPong, func(msg Message) error {
		n.routing.addContact(msg.FromContact)
		return nil
	})

	n.Handle("find_node", func(msg Message) error {
		key := string(msg.Payload)
		closest := n.routing.getKClosest(key, K)
		return n.Send(msg.From, "find_node_response", []byte(tripleSerialize(closest)))
	})

	n.Handle("find_value", func(msg Message) error {
		key := string(msg.Payload)
		n.routing.addContact(msg.FromContact)

		if val, ok := n.FindObjectLocally(key); ok {
			return n.Send(msg.From, "find_value_response", []byte("VALUE:"+string(val)))
		}

		closest := n.routing.getKClosest(key, K)
		return n.Send(msg.From, "find_value_response", []byte(tripleSerialize(closest)))
	})
}

// Utility functions
func xorDistance(a, b []byte) *big.Int {
	return new(big.Int).Xor(new(big.Int).SetBytes(a), new(big.Int).SetBytes(b))
}

func tripleSerialize(triples []Triple) string {
	var parts []string
	for _, t := range triples {
		parts = append(parts, fmt.Sprintf("%s:%x", t.Addr.String(), t.ID))
	}
	return strings.Join(parts, ",")
}

func tripleDeserialize(s string) ([]Triple, error) {
	var triples []Triple
	for _, part := range strings.Split(s, ",") {
		if part == "" {
			continue
		}
		segments := strings.Split(part, ":")
		if len(segments) >= 3 {
			port, _ := strconv.Atoi(segments[1])
			idHex := segments[2]
			idBytes := make([]byte, len(idHex)/2)
			for i := 0; i < len(idHex); i += 2 {
				b, _ := strconv.ParseUint(idHex[i:i+2], 16, 8)
				idBytes[i/2] = byte(b)
			}
			triples = append(triples, Triple{
				ID:   idBytes,
				Addr: Address{IP: segments[0], Port: port},
				Port: port,
			})
		}
	}
	return triples, nil
}

func (n *Node) Close() error {
	n.closeMu.Lock()
	n.closed = true
	n.closeMu.Unlock()
	return n.connection.Close()
}

func (n *Node) Address() Address {
	return n.addr
}
