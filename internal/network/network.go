package network

import (
    "fmt"
    "net"
    "time"

    . "github.com/eislab-cps/go-template/pkg/kademlia"
)

// Kademlia message types
const (
    MsgPing      = "PING"
    MsgPong      = "PONG"
    MsgFindNode  = "FIND_NODE"
    MsgFindValue = "FIND_VALUE"
    MsgStore     = "STORE"
)

// Network defines the interface for network operations
type Network interface {
    Listen(addr Address) (Connection, error)
    Dial(addr Address) (Connection, error)
    // Network partition simulation (keep for testing)
    Partition(group1, group2 []Address)
    Heal()
}

// Connection defines the interface for network connections
type Connection interface {
    Send(msg Message) error
    Recv() (Message, error)
    Close() error
    LocalAddr() net.Addr
}

// Message represents a network message
type Message struct {
    ID          [20]byte
    From        Address
    FromContact Triple
    To          Address
    Payload     []byte
    Network     Network
}

// UDPNetwork implements real UDP networking
type UDPNetwork struct {
    partitioned bool
    partition1  map[string]bool
    partition2  map[string]bool
}

// NewUDPNetwork creates a new UDP network instance
func NewUDPNetwork() *UDPNetwork {
    return &UDPNetwork{
        partitioned: false,
        partition1:  make(map[string]bool),
        partition2:  make(map[string]bool),
    }
}

// Listen creates a listening UDP connection
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
        conn:      conn,
        localAddr: addr,
        network:   n,
        connected: false,
    }, nil
}

// Dial creates a connected UDP connection
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

    // Connected UDP socket to that remote (one-off sender)
    conn, err := net.DialUDP("udp", nil, udpAddr)
    if err != nil {
        return nil, fmt.Errorf("failed to dial %s: %v", addr.String(), err)
    }

    // Set timeout for operations (one-off)
    _ = conn.SetDeadline(time.Now().Add(5 * time.Second))

    return &UDPConnection{
        conn:      conn,
        localAddr: AddressFromNetAddr(conn.LocalAddr()),
        remote:    udpAddr,
        network:   n,
        connected: true,
    }, nil
}

// Partition simulates network partition for testing
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

// Heal removes network partition
func (n *UDPNetwork) Heal() {
    n.partitioned = false
    n.partition1 = make(map[string]bool)
    n.partition2 = make(map[string]bool)
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
    connection, err := m.Network.Dial(m.From)
    if err != nil {
        return fmt.Errorf("failed to dial %s: %v", m.From.String(), err)
    }
    defer connection.Close()

    // Create reply message
    reply := Message{
        From:    m.To,
        To:      m.From,
        Payload: payload,
        Network: m.Network,
    }

    return connection.Send(reply)
}

// ReplyString is a convenience method for sending string replies
func (m Message) ReplyString(msgType, data string) error {
    return m.Reply(msgType, []byte(data))
}