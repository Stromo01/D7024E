package main

import "fmt"

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
	From    Address
	To      Address
	Payload []byte
	network Network // Reference to network for replies
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
