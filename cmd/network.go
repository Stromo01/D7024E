package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
)

// Kademlia message types
const (
	MsgPing      = "PING"
	MsgPong      = "PONG"
	MsgStore     = "STORE"
	MsgFindNode  = "FIND_NODE"
	MsgFindValue = "FIND_VALUE"
)

// Response types
const (
	MsgPongResp      = "PONG_RESP"
	MsgStoreResp     = "STORE_RESP"
	MsgFindNodeResp  = "FIND_NODE_RESP"
	MsgFindValueResp = "FIND_VALUE_RESP"
)

// RPC represents a Kademlia RPC message
type RPC struct {
	Type    string      `json:"type"`
	ID      [20]byte    `json:"id"`      // 160-bit RPC identifier
	Payload interface{} `json:"payload"` // Varies by message type
}

// StoreRequest represents a STORE RPC request
type StoreRequest struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

// StoreResponse represents a STORE RPC response
type StoreResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// FindNodeRequest represents a FIND_NODE RPC request
type FindNodeRequest struct {
	Target []byte `json:"target"` // 160-bit key we're looking for
}

// FindNodeResponse represents a FIND_NODE RPC response
type FindNodeResponse struct {
	Contacts []Triple `json:"contacts"` // Up to K closest contacts
}

// FindValueRequest represents a FIND_VALUE RPC request
type FindValueRequest struct {
	Key string `json:"key"`
}

// FindValueResponse represents a FIND_VALUE RPC response
type FindValueResponse struct {
	Found    bool     `json:"found"`
	Value    []byte   `json:"value,omitempty"`    // If found
	Contacts []Triple `json:"contacts,omitempty"` // If not found
}

// Helper to send a PING message from one node to another
func SendPing(network Network, from, to Address) error {
	// Use the standard SendRPC function for consistency
	_, err := SendRPC(network, from, to, MsgPing, nil)
	return err
} // SendRPC sends an RPC message and returns the RPC ID for tracking responses
func SendRPC(network Network, from, to Address, rpcType string, payload interface{}) ([20]byte, error) {
	var rpcID [20]byte
	rand.Read(rpcID[:])

	rpc := RPC{
		Type:    rpcType,
		ID:      rpcID,
		Payload: payload,
	}

	data, err := json.Marshal(rpc)
	if err != nil {
		return rpcID, err
	}

	// Format as "msgType:jsonPayload" to work with existing message system
	msgPayload := append([]byte(rpcType+":"), data...)

	msg := Message{
		From:    from,
		To:      to,
		Payload: msgPayload,
		network: network,
		FromContact: Triple{
			ID:   make([]byte, 20), // Will be set by actual node
			Addr: from,
			Port: from.Port,
		},
	}

	conn, err := network.Dial(to)
	if err != nil {
		return rpcID, err
	}
	defer conn.Close()

	return rpcID, conn.Send(msg)
}

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

	// Network partition simulation
	Partition(group1, group2 []Address)
	Heal()
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	From        Address
	FromContact Triple
	To          Address
	Payload     []byte
	network     Network // Reference to network for replies
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

	// Make sure we have a network reference
	if m.network == nil {
		return fmt.Errorf("cannot reply: message has no network reference")
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

// ReplyRPC sends an RPC response back to the sender
func (m Message) ReplyRPC(originalRPCID [20]byte, responseType string, payload interface{}) error {
	rpc := RPC{
		Type:    responseType,
		ID:      originalRPCID, // Echo the original RPC ID
		Payload: payload,
	}

	data, err := json.Marshal(rpc)
	if err != nil {
		return err
	}

	// Format as "msgType:jsonPayload" to work with existing message system (like SendRPC does)
	msgPayload := append([]byte(responseType+":"), data...)

	// Make sure we have a network reference
	if m.network == nil {
		return fmt.Errorf("cannot reply: message has no network reference")
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
		Payload: msgPayload,
		network: m.network,
	}

	return connection.Send(reply)
}

// ReplyString is a convenience method for sending string replies
func (m Message) ReplyString(data string) error {
	return m.Reply("", []byte(data))
}

// ReplyStringWithType is a convenience method for sending string replies with message type
func (m Message) ReplyStringWithType(msgType, data string) error {
	return m.Reply(msgType, []byte(data))
}
