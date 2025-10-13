package network

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// UDPConnection serves both listening and connected UDP sockets
type UDPConnection struct {
	conn      *net.UDPConn
	localAddr Address
	connected bool
	remote    *net.UDPAddr
	network   *UDPNetwork
	sendMu    sync.Mutex
}

// Send transmits a message through the UDP connection
func (c *UDPConnection) Send(msg Message) error {
	w := WireMessage{ID: msg.ID, Type: msg.Type, FromContact: msg.FromContact, Payload: msg.Payload}
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	if c.connected {
		_, err = c.conn.Write(b)
		return err
	}
	ra, err := net.ResolveUDPAddr("udp", msg.To.String())
	if err != nil {
		return fmt.Errorf("resolve dst: %w", err)
	}
	_, err = c.conn.WriteToUDP(b, ra)
	return err
}

// Recv receives a message from the UDP connection
func (c *UDPConnection) Recv() (Message, error) {
	buf := make([]byte, 65535)
	if c.connected {
		n, err := c.conn.Read(buf)
		if err != nil {
			return Message{}, err
		}
		var w WireMessage
		if err := json.Unmarshal(buf[:n], &w); err != nil {
			return Message{}, err
		}
		return Message{
			ID:          w.ID,
			Type:        w.Type,
			From:        AddressFromNetAddr(c.conn.RemoteAddr()),
			FromContact: w.FromContact,
			To:          c.localAddr,
			Payload:     w.Payload,
			Network:     c.network,
		}, nil
	}
	n, ra, err := c.conn.ReadFromUDP(buf)
	if err != nil {
		return Message{}, err
	}
	var w WireMessage
	if err := json.Unmarshal(buf[:n], &w); err != nil {
		return Message{}, err
	}
	return Message{
		ID:          w.ID,
		Type:        w.Type,
		From:        Address{IP: ra.IP.String(), Port: ra.Port},
		FromContact: w.FromContact,
		To:          c.localAddr,
		Payload:     w.Payload,
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
