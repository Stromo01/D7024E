package node

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

type Node struct {
	Id         [20]byte // 160 bits
	Addr       Address
	network    Network
	connection Connection
	handlers   map[string]MessageHandler
	store      map[string][]byte // Object store: key (hash) -> value
	routing    *RoutingTable
	mu         sync.RWMutex
	closed     bool
	closeMu    sync.RWMutex
	pending    map[[20]byte]chan Message
	pendingMu  sync.Mutex

	messageWG     sync.WaitGroup // Tracks active message handlers
	maxConcurrent int            // Maximum concurrent handlers
	semaphore     chan struct{}  // Semaphore to limit concurrency
}

const K = 3     // Kademlia bucket size and number of closest nodes to return
const Alpha = 3 // Concurrency in lookups

// XOR distance between two keys (as hex strings)
func XorDistance(a, b []byte) *big.Int {
	aInt := new(big.Int).SetBytes(a)
	bInt := new(big.Int).SetBytes(b)
	return new(big.Int).Xor(aInt, bInt)
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
		return value, n.Addr.String(), true
	}
	fmt.Print("Object not found locally")

	// Use iterative find value for network lookup
	if value, found := n.iterativeFindValue(key); found {
		return value, "network", true
	}
	fmt.Print("Object not found in network")
	return nil, "", false
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg Message) error

// NewNode creates a new node that can both send and receive messages
func NewNode(network Network, addr Address) (*Node, error) {
	bindAddr := Address{IP: "0.0.0.0", Port: addr.Port}

	connection, err := network.Listen(bindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}

	actualAddr := AddressFromNetAddr(connection.LocalAddr())
	advertiseAddr := addr
	if addr.IP == "" || addr.IP == "0.0.0.0" {
		advertiseAddr = actualAddr
	}

	var id [20]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, fmt.Errorf("failed to generate node ID: %v", err)
	}

	maxConcurrent := 10 // Maximum concurrent handlers

	node := &Node{
		Id:            id,
		Addr:          advertiseAddr,
		network:       network,
		connection:    connection,
		handlers:      make(map[string]MessageHandler),
		routing:       NewRoutingTable(Triple{ID: id[:], Addr: advertiseAddr, Port: advertiseAddr.Port}),
		store:         make(map[string][]byte),
		pending:       make(map[[20]byte]chan Message),
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}

	node.registerHandlers() // Register all message handlers
	return node, nil
}
func (n *Node) registerHandlers() {
	n.Handle("store", n.handleStore)
	n.Handle(MsgPing, n.handlePing)
	n.Handle(MsgPong, n.handlePong)
	n.Handle("find_node", n.handleFindNode)
	n.Handle("find_node_response", n.handleFindNodeResponse)
	n.Handle("find_value", n.handleFindValue)
	n.Handle("find_value_response", n.handleFindValueResponse)
}

func (n *Node) GetAllContacts() []Triple {
	var contacts []Triple
	for _, bucket := range n.routing.Buckets {
		if bucket != nil {
			// Use GetAllContacts() method from bucket
			contacts = append(contacts, bucket.GetAllContacts()...)
		}
	}
	return contacts
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
	err = n.Send(bootstrapNode.Addr, "find_node", []byte(fmt.Sprintf("%x", n.Id)))
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
	if n == nil {
		return
	}
	if n.connection == nil {
		log.Println("node.Start: no connection set, returning")
		return
	}

	for {
		// check closed flag
		n.closeMu.RLock()
		if n.closed {
			n.closeMu.RUnlock()
			return
		}
		n.closeMu.RUnlock()

		// Blocking receive
		msg, err := n.connection.Recv()
		if err != nil {
			n.closeMu.RLock()
			if !n.closed {
				log.Printf("Node %s failed to receive message: %v", n.Addr.String(), err)
			}
			n.closeMu.RUnlock()
			break
		}

		n.HandleMessageConcurrently(msg)
	}
	log.Println("Waiting for active message handlers to finish...")
	n.messageWG.Wait()
	log.Println("Node closed successfully")
}

// Send sends a message to the target address
func (n *Node) Send(to Address, msgType string, data []byte) error {
	msg := Message{
		From:        n.Addr,
		FromContact: Triple{ID: n.Id[:], Addr: n.Addr, Port: n.Addr.Port},
		To:          to,
		Type:        msgType,
		Payload:     data,
		Network:     n.network,
	}

	return n.connection.Send(msg)
}

func (n *Node) Close() error {
	log.Println("Closing node...")

	// Set closed flag to stop receiving new messages
	n.closeMu.Lock()
	n.closed = true
	n.closeMu.Unlock()

	// Close connection to interrupt Recv()
	err := n.connection.Close()

	// Wait for all active message handlers to complete
	log.Println("Waiting for active message handlers to finish...")
	n.messageWG.Wait()
	log.Println("Node closed successfully")

	return err
}

// Address returns the node's address
func (n *Node) Address() Address {
	return n.Addr
}

func (n *Node) RoutingTable() *RoutingTable {
	return n.routing
}
