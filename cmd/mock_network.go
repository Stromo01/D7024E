package main

import (
	"errors"
	"sync"
)

type mockNetwork struct {
	mu         sync.RWMutex
	listeners  map[Address]chan Message
	partitions map[Address]map[Address]bool // map of address to unreachable addresses
}

func NewMockNetwork() Network {
	return &mockNetwork{
		listeners:  make(map[Address]chan Message),
		partitions: make(map[Address]map[Address]bool),
	}
}

func (n *mockNetwork) Listen(addr Address) (Connection, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.listeners[addr]; exists {
		return nil, errors.New("address already in use")
	}
	ch := make(chan Message, 100) // buffered channel
	n.listeners[addr] = ch
	return &mockConnection{addr: addr, network: n, recvCh: ch}, nil
}

func (n *mockNetwork) Dial(addr Address) (Connection, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Check if the destination address exists
	if _, exists := n.listeners[addr]; !exists {
		return nil, errors.New("address not found")
	}

	// Return a connection that will check partitioning during Send operations
	return &mockConnection{addr: addr, network: n}, nil
}

func (n *mockNetwork) Partition(group1, group2 []Address) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Initialize or reset the partitions map
	n.partitions = make(map[Address]map[Address]bool)

	// For each address in group1, add all addresses in group2 to its unreachable list
	for _, addr1 := range group1 {
		if n.partitions[addr1] == nil {
			n.partitions[addr1] = make(map[Address]bool)
		}
		for _, addr2 := range group2 {
			n.partitions[addr1][addr2] = true
		}
	}

	// For each address in group2, add all addresses in group1 to its unreachable list
	for _, addr2 := range group2 {
		if n.partitions[addr2] == nil {
			n.partitions[addr2] = make(map[Address]bool)
		}
		for _, addr1 := range group1 {
			n.partitions[addr2][addr1] = true
		}
	}
}

func (n *mockNetwork) Heal() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.partitions = make(map[Address]map[Address]bool)
}

type mockConnection struct {
	addr    Address
	network *mockNetwork
	recvCh  chan Message
	mu      sync.RWMutex
	closed  bool
}

func (c *mockConnection) Send(msg Message) error {
	c.network.mu.RLock()
	defer c.network.mu.RUnlock()

	// Check if the sender is partitioned from the destination
	if partitions, exists := c.network.partitions[msg.From]; exists && partitions[msg.To] {
		return errors.New("network partitioned")
	}

	ch, exists := c.network.listeners[msg.To]
	if !exists {
		return errors.New("destination address not found")
	}

	// Add network reference to message for replies
	msg.network = c.network

	// Try to send the message
	select {
	case ch <- msg:
		return nil
	default:
		return errors.New("message queue full")
	}
}

func (c *mockConnection) Recv() (Message, error) {
	c.mu.RLock()
	if c.closed || c.recvCh == nil {
		c.mu.RUnlock()
		return Message{}, errors.New("connection not listening")
	}
	ch := c.recvCh
	c.mu.RUnlock()

	msg, ok := <-ch
	if !ok {
		return Message{}, errors.New("connection closed")
	}
	return msg, nil
}

func (c *mockConnection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil // Already closed
	}
	c.closed = true
	c.mu.Unlock()

	c.network.mu.Lock()
	defer c.network.mu.Unlock()

	if c.recvCh != nil {
		close(c.recvCh)
		delete(c.network.listeners, c.addr)
		c.recvCh = nil
	}
	return nil
}
