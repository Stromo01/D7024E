package network

import (
	"fmt"
	"net"

	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// UDPConnection serves both listening and connected UDP sockets
type UDPConnection struct {
	conn      *net.UDPConn
	localAddr Address
	connected bool
	remote    *net.UDPAddr
	network   *UDPNetwork
}

// Send transmits a message through the UDP connection
func (c *UDPConnection) Send(msg Message) error {
	wire, err := encodeWireMessage(WireMessage{
		FromContact: msg.FromContact,
		Type:        msg.Type,
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

// Recv receives a message from the UDP connection
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
			Type:        wire.Type,
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
		Type:        wire.Type,
		Network:     c.network,
	}, nil
}

// Close closes the UDP connection
func (c *UDPConnection) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// LocalAddr returns the local network address
func (c *UDPConnection) LocalAddr() net.Addr {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.LocalAddr()
}
