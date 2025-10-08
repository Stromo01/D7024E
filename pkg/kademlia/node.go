package kademlia

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
)

// Constants
const K = 8     // Kademlia bucket size and number of closest nodes to return
const Alpha = 3 // Concurrency in lookups

// Types that need to be defined or imported
type Address struct {
	IP   string
	Port int
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

type Network interface {
	Listen(addr Address) (Connection, error)
	Dial(addr Address) (Connection, error)
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	From    Address
	To      Address
	Payload []byte
}

type MessageHandler func(msg Message) error

type Triple struct {
	ID   []byte
	Addr Address
	Port int
}

// RoutingTable placeholder - you'll need to implement this
type RoutingTable struct {
	// Implementation needed
}

func NewRoutingTable(self Triple) *RoutingTable {
	// TODO: Implement
	return &RoutingTable{}
}

func (rt *RoutingTable) getKClosest(key string, k int) []Triple {
	// TODO: Implement
	return []Triple{}
}

// Node struct
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

// XOR distance between two keys
func XorDistance(a, b []byte) *big.Int {
	aInt := new(big.Int).SetBytes(a)
	bInt := new(big.Int).SetBytes(b)
	return new(big.Int).Xor(aInt, bInt)
}

// NewNode creates a new node
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
	return node, nil
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

// IterativeFindNode - exported version of iterativeFindNode
func (n *Node) IterativeFindNode(key string) []Triple {
	// TODO: Implement the actual logic from your original method
	return []Triple{}
}

// NodeLookup - placeholder for the nodeLookup method
func (n *Node) NodeLookup(key string) []Triple {
	// TODO: Implement
	return []Triple{}
}

// Send sends a message to the target address
func (n *Node) Send(to Address, msgType string, data []byte) error {
	connection, err := n.network.Dial(to)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", to.String(), err)
	}
	defer connection.Close()

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

// Address returns the node's address
func (n *Node) Address() Address {
	return n.addr
}

// Close shuts down the node
func (n *Node) Close() error {
	n.closeMu.Lock()
	n.closed = true
	n.closeMu.Unlock()
	return n.connection.Close()
}
