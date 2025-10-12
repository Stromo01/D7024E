package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Kademlia message types
const (
	MsgPing      = "PING"
	MsgPong      = "PONG"
	MsgFindNode  = "FIND_NODE"
	MsgFindValue = "FIND_VALUE"
	MsgStore     = "STORE"
)

type Address struct {
	IP   string
	Port int // 1-65535
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

type Network interface {
	Listen(addr Address) (Connection, error)
	Dial(addr Address) (Connection, error)
	// Network partition simulation (keep for testing)
	Partition(group1, group2 []Address)
	Heal()
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	ID          [20]byte
	From        Address
	FromContact Triple
	To          Address
	Payload     []byte
	network     Network // Reference to network for replies
}

// UDPNetwork implements real UDP networking
type UDPNetwork struct {
	partitioned bool
	partition1  map[string]bool
	partition2  map[string]bool
}

func NewUDPNetwork() *UDPNetwork {
	return &UDPNetwork{
		partitioned: false,
		partition1:  make(map[string]bool),
		partition2:  make(map[string]bool),
	}
}

func (n *UDPNetwork) Listen(addr Address) (Connection, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address %s: %v", addr.String(), err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %v", addr.String(), err)
	}

	return &UDPConnection{
		conn:    conn,
		addr:    addr,
		network: n,
	}, nil
}

func (n *UDPNetwork) Dial(addr Address) (Connection, error) {
	// Check for network partition
	if n.partitioned {
		if n.partition1[addr.String()] || n.partition2[addr.String()] {
			return nil, fmt.Errorf("network partitioned")
		}
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address %s: %v", addr.String(), err)
	}

	// IMPORTANT: Don't use DialUDP - it creates random source ports
	// Instead, create a temporary connection for sending
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", addr.String(), err)
	}

	// Set timeout for operations
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	return &UDPDialConnection{
		conn:    conn,
		network: n,
	}, nil
}

func (n *UDPNetwork) Partition(group1, group2 []Address) {
	n.partitioned = true
	n.partition1 = make(map[string]bool)
	n.partition2 = make(map[string]bool)

	for _, addr := range group1 {
		n.partition1[addr.String()] = true
	}
	for _, addr := range group2 {
		n.partition2[addr.String()] = true
	}
}

func (n *UDPNetwork) Heal() {
	n.partitioned = false
	n.partition1 = make(map[string]bool)
	n.partition2 = make(map[string]bool)
}

// UDPConnection for listening connections
type UDPConnection struct {
	conn    *net.UDPConn
	addr    Address
	network *UDPNetwork
}

func (c *UDPConnection) Send(msg Message) error {
	// Serialize the message for transmission
	wireMsg := WireMessage{
		FromContact: msg.FromContact,
		Payload:     msg.Payload,
	}

	data, err := json.Marshal(wireMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// Resolve target address
	targetAddr, err := net.ResolveUDPAddr("udp", msg.To.String())
	if err != nil {
		return fmt.Errorf("failed to resolve target address: %v", err)
	}

	// Send using the listening connection (maintains source port)
	_, err = c.conn.WriteToUDP(data, targetAddr)
	if err != nil {
		return fmt.Errorf("failed to write UDP message: %v", err)
	}

	fmt.Printf("Node %s sent message to %s\n", c.addr.String(), msg.To.String())
	return nil
}

func (c *UDPConnection) Recv() (Message, error) {
	buffer := make([]byte, 4096)
	n, remoteAddr, err := c.conn.ReadFromUDP(buffer)
	if err != nil {
		return Message{}, fmt.Errorf("failed to read UDP message: %v", err)
	}

	// Parse the remote address correctly
	fromAddr := Address{
		IP:   remoteAddr.IP.String(),
		Port: remoteAddr.Port,
	}

	// Deserialize the message
	var wireMsg WireMessage
	if err := json.Unmarshal(buffer[:n], &wireMsg); err != nil {
		return Message{}, fmt.Errorf("failed to unmarshal message: %v", err)
	}

	return Message{
		From:        fromAddr,
		FromContact: wireMsg.FromContact,
		To:          c.addr,
		Payload:     wireMsg.Payload,
		network:     c.network,
	}, nil
}

func (c *UDPConnection) Close() error {
	return c.conn.Close()
}

// UDPDialConnection for outgoing connections
type UDPDialConnection struct {
	conn    *net.UDPConn
	network *UDPNetwork
}

func (c *UDPDialConnection) Send(msg Message) error {
	// Serialize the message for transmission
	wireMsg := WireMessage{
		FromContact: msg.FromContact,
		Payload:     msg.Payload,
	}

	data, err := json.Marshal(wireMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	_, err = c.conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write UDP message: %v", err)
	}

	return nil
}

func (c *UDPDialConnection) Recv() (Message, error) {
	// This shouldn't typically be called on dial connections
	// But implement for completeness
	buffer := make([]byte, 4096)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return Message{}, fmt.Errorf("failed to read UDP message: %v", err)
	}

	// For dial connections, we know the remote address from the connection
	remoteAddr := c.conn.RemoteAddr().(*net.UDPAddr)
	fromAddr := Address{
		IP:   remoteAddr.IP.String(),
		Port: remoteAddr.Port,
	}

	// Deserialize the message
	var wireMsg WireMessage
	if err := json.Unmarshal(buffer[:n], &wireMsg); err != nil {
		return Message{}, fmt.Errorf("failed to unmarshal message: %v", err)
	}

	return Message{
		From:        fromAddr,
		FromContact: wireMsg.FromContact,
		To:          Address{}, // We don't know our own address in this context
		Payload:     wireMsg.Payload,
		network:     c.network,
	}, nil
}

func (c *UDPDialConnection) Close() error {
	return c.conn.Close()
}

// WireMessage is the serializable format for network transmission
type WireMessage struct {
	FromContact Triple `json:"from_contact"`
	Payload     []byte `json:"payload"`
}

// Helper functions

// GetLocalIP returns the local IP address
func GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// Fallback: try to find eth0 or other interfaces
		return getIPFromInterface()
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func getIPFromInterface() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Try eth0 first, then fall back to other interfaces
	for _, i := range interfaces {
		if i.Name == "eth0" {
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String(), nil
					}
				}
			}
		}
	}

	// Fallback to any non-loopback interface
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
	}

	return "127.0.0.1", nil // Ultimate fallback
}

// SendPing helper function
func SendPing(network Network, from, to Address) error {
	conn, err := network.Dial(to)
	if err != nil {
		return err
	}
	defer conn.Close()

	msg := Message{
		From:    from,
		To:      to,
		Payload: []byte(MsgPing + ":ping"),
		network: network,
	}

	return conn.Send(msg)
}

// Reply sends a response message back to the sender
func (m Message) Reply(msgType string, data []byte) error {
	// Format payload as "msgType:data"
	var payload []byte
	if msgType != "" {
		payload = append([]byte(msgType+":"), data...)
	} else {
		payload = data
	}

	// Create connection to sender
	connection, err := m.network.Dial(m.From)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", m.From.String(), err)
	}
	defer connection.Close()

	// Create reply message
	reply := Message{
		From:    m.To,
		To:      m.From,
		Payload: payload,
		network: m.network,
	}

	return connection.Send(reply)
}

// ReplyString is a convenience method for sending string replies
func (m Message) ReplyString(msgType, data string) error {
	return m.Reply(msgType, []byte(data))
}
