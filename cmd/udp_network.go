package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

// UDPNetwork implements the Network interface using real UDP sockets
type UDPNetwork struct {
	connections map[string]*UDPConnection
	mu          sync.RWMutex
}

// UDPConnection wraps a UDP connection for our Network interface
type UDPConnection struct {
	conn       *net.UDPConn
	localAddr  Address
	remoteAddr Address
	network    *UDPNetwork
}

// NewUDPNetwork creates a new UDP network
func NewUDPNetwork() *UDPNetwork {
	return &UDPNetwork{
		connections: make(map[string]*UDPConnection),
	}
}

// Listen creates a UDP listener on the specified address
func (n *UDPNetwork) Listen(addr Address) (Connection, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr.IP, addr.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %v", err)
	}

	udpConn := &UDPConnection{
		conn:      conn,
		localAddr: addr,
		network:   n,
	}

	key := fmt.Sprintf("%s:%d", addr.IP, addr.Port)
	n.mu.Lock()
	n.connections[key] = udpConn
	n.mu.Unlock()

	return udpConn, nil
}

// Dial creates a connection to a remote address
func (n *UDPNetwork) Dial(target Address) (Connection, error) {
	// For UDP, we don't really "dial", but we create a connection object
	// that knows the target address
	localAddr := Address{IP: "0.0.0.0", Port: 0} // Let system choose local port

	udpConn := &UDPConnection{
		localAddr:  localAddr,
		remoteAddr: target,
		network:    n,
	}

	return udpConn, nil
}

// Partition and Heal are for testing - not needed for real UDP
func (n *UDPNetwork) Partition(group1, group2 []Address) {
	// Not applicable for real UDP network
}

func (n *UDPNetwork) Heal() {
	// Not applicable for real UDP network
}

// Send implements the Connection interface
func (c *UDPConnection) Send(msg Message) error {
	// Serialize the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if c.conn == nil {
		// This is a "dial" connection, create a temporary connection
		localAddr, err := net.ResolveUDPAddr("udp", ":0")
		if err != nil {
			return err
		}

		conn, err := net.DialUDP("udp", localAddr, &net.UDPAddr{
			IP:   net.ParseIP(c.remoteAddr.IP),
			Port: c.remoteAddr.Port,
		})
		if err != nil {
			return err
		}
		defer conn.Close()

		_, err = conn.Write(data)
		return err
	}

	// This is a listener connection, send to the remote address
	remoteAddr := &net.UDPAddr{
		IP:   net.ParseIP(c.remoteAddr.IP),
		Port: c.remoteAddr.Port,
	}

	_, err = c.conn.WriteToUDP(data, remoteAddr)
	return err
}

// Recv implements the Connection interface
func (c *UDPConnection) Recv() (Message, error) {
	if c.conn == nil {
		return Message{}, fmt.Errorf("cannot receive on dial connection")
	}

	buffer := make([]byte, 65536) // Max UDP packet size
	n, addr, err := c.conn.ReadFromUDP(buffer)
	if err != nil {
		return Message{}, err
	}

	// Update remote address for potential replies
	c.remoteAddr = Address{
		IP:   addr.IP.String(),
		Port: addr.Port,
	}

	// Deserialize the message
	var msg Message
	err = json.Unmarshal(buffer[:n], &msg)
	if err != nil {
		return Message{}, err
	}

	return msg, nil
}

// Close implements the Connection interface
func (c *UDPConnection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
