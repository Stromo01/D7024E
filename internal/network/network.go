package network

import (
	"encoding/json"
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
	LocalAddr() net.Addr
}

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
		conn:      conn,
		localAddr: addr,
		network:   n,
		connected: false,
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

// UDPConnection serves both listening and connected UDP sockets.
type UDPConnection struct {
	conn      *net.UDPConn
	localAddr Address

	// For connected sockets (Dial)
	connected bool
	remote    *net.UDPAddr

	network *UDPNetwork
}

func (c *UDPConnection) Send(msg Message) error {
	wire, err := encodeWireMessage(WireMessage{
		FromContact: msg.FromContact,
		Payload:     msg.Payload,
	})
	if err != nil {
		return err
	}

	if c.connected {
		// Connected socket: destination is fixed
		if _, err := c.conn.Write(wire); err != nil {
			return fmt.Errorf("failed to write UDP message: %v", err)
		}
		return nil
	}

	// Listening socket: send to msg.To
	targetAddr, err := net.ResolveUDPAddr("udp", msg.To.String())
	if err != nil {
		return fmt.Errorf("failed to resolve target address: %v", err)
	}
	if _, err := c.conn.WriteToUDP(wire, targetAddr); err != nil {
		return fmt.Errorf("failed to write UDP message: %v", err)
	}
	return nil
}

func (c *UDPConnection) Recv() (Message, error) {
	buffer := make([]byte, 4096)

	if c.connected {
		// Connected socket read
		n, err := c.conn.Read(buffer)
		if err != nil {
			return Message{}, fmt.Errorf("failed to read UDP message: %v", err)
		}
		wire, err := decodeWireMessage(buffer[:n])
		if err != nil {
			return Message{}, err
		}
		from := AddressFromNetAddr(c.conn.RemoteAddr())
		return Message{
			From:        from,
			FromContact: wire.FromContact,
			To:          c.localAddr,
			Payload:     wire.Payload,
			Network:     c.network,
		}, nil
	}

	// Listening socket read
	n, remoteAddr, err := c.conn.ReadFromUDP(buffer)
	if err != nil {
		return Message{}, fmt.Errorf("failed to read UDP message: %v", err)
	}
	wire, err := decodeWireMessage(buffer[:n])
	if err != nil {
		return Message{}, err
	}
	from := Address{
		IP:   remoteAddr.IP.String(),
		Port: remoteAddr.Port,
	}
	return Message{
		From:        from,
		FromContact: wire.FromContact,
		To:          c.localAddr,
		Payload:     wire.Payload,
		Network:     c.network,
	}, nil
}

func (c *UDPConnection) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *UDPConnection) LocalAddr() net.Addr {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.LocalAddr()
}

// WireMessage is the serializable format for network transmission
type WireMessage struct {
	FromContact Triple `json:"from_contact"`
	Payload     []byte `json:"payload"`
}

// Helpers

func encodeWireMessage(m WireMessage) ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %v", err)
	}
	return data, nil
}

func decodeWireMessage(b []byte) (WireMessage, error) {
	var m WireMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return WireMessage{}, fmt.Errorf("failed to unmarshal message: %v", err)
	}
	return m, nil
}

func AddressFromNetAddr(a net.Addr) Address {
	if a == nil {
		return Address{}
	}
	if ua, ok := a.(*net.UDPAddr); ok {
		return Address{IP: ua.IP.String(), Port: ua.Port}
	}
	// Fallback parse host:port
	host, portStr, err := net.SplitHostPort(a.String())
	if err != nil {
		return Address{}
	}
	p := 0
	fmt.Sscanf(portStr, "%d", &p)
	return Address{IP: host, Port: p}
}

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
		Network: network,
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
