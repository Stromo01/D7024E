package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"math/big"
	"sort"
	"sync"
)

// XOR distance between two keys (as hex strings)
// Removed duplicate definition of xorDistance

type Node struct {
	addr       Address
	network    Network
	connection Connection
	handlers   map[string]MessageHandler
	contacts   []Address         // List of known contacts
	store      map[string][]byte // Object store: key (hash) -> value
	mu         sync.RWMutex
	closed     bool
	closeMu    sync.RWMutex
}

// XOR distance between two keys (as hex strings)
func xorDistance(a, b string) *big.Int {
	aBytes := sha1.Sum([]byte(a))
	bBytes := sha1.Sum([]byte(b))
	aInt := new(big.Int).SetBytes(aBytes[:])
	bInt := new(big.Int).SetBytes(bBytes[:])
	return new(big.Int).Xor(aInt, bInt)
}

// FindKClosest returns the K closest contacts to the given key
func (n *Node) FindKClosest(key string, k int) []Address {
	n.mu.RLock()
	defer n.mu.RUnlock()
	type distAddr struct {
		dist *big.Int
		addr Address
	}
	var all []distAddr
	for _, c := range n.contacts {
		d := xorDistance(key, c.String())
		all = append(all, distAddr{dist: d, addr: c})
	}
	// Sort by distance
	sort.Slice(all, func(i, j int) bool {
		return all[i].dist.Cmp(all[j].dist) < 0
	})
	var result []Address
	for i := 0; i < k && i < len(all); i++ {
		result = append(result, all[i].addr)
	}
	return result
}

// StoreAtK stores an object at the K closest nodes (including self if applicable)
func (n *Node) StoreAtK(key string, value []byte, k int) {
	closest := n.FindKClosest(key, k)
	for _, addr := range closest {
		if addr == n.addr {
			n.StoreObject(key, value)
		} else {
			payload := []byte(key + ":" + string(value))
			n.Send(addr, "store", payload)
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

// FindNode returns the contact Address if present, else nil
func (n *Node) FindNode(addr Address) *Address {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, c := range n.contacts {
		if c == addr {
			return &c
		}
	}
	return nil
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg Message) error

// NewNode creates a new node that can both send and receive messages
func NewNode(network Network, addr Address) (*Node, error) {
	connection, err := network.Listen(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}

	node := &Node{
		addr:       addr,
		network:    network,
		connection: connection,
		handlers:   make(map[string]MessageHandler),
		contacts:   []Address{},
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
		node.addContact(msg.From)
		fmt.Printf("Node %s received PING from %s\n", node.Address().String(), msg.From.String())
		return node.Send(msg.From, MsgPong, []byte("pong"))
	})

	// Register PONG handler (for logging and adding contact)
	node.Handle(MsgPong, func(msg Message) error {
		node.addContact(msg.From)
		fmt.Printf("Node %s received PONG from %s\n", node.Address().String(), msg.From.String())
		return nil
	})

	return node, nil
}

// Add contact if not already present
func (n *Node) addContact(addr Address) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, c := range n.contacts {
		if c == addr {
			return
		}
	}
	n.contacts = append(n.contacts, addr)
}

// JoinNetwork: send PING to known node and add to contacts
func (n *Node) JoinNetwork(known Address) error {
	err := n.Send(known, MsgPing, []byte("ping"))
	if err != nil {
		return fmt.Errorf("failed to join network: %v", err)
	}
	// Contact will be added when PONG is received
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
