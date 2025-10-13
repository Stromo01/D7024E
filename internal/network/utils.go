package network

import (
	"encoding/json"
	"fmt"
	"net"

	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// WireMessage is the serializable format for network transmission
type WireMessage struct {
	FromContact Triple `json:"from_contact"`
	Payload     []byte `json:"payload"`
}

// Message encoding/decoding functions
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

// AddressFromNetAddr converts a net.Addr to Address
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
