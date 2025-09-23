package kademlia

// Message represents a Kademlia message
type Message struct {
	From    Address
	To      Address
	Payload []byte
}

// Network interface for Kademlia
type Network interface {
	Listen(addr Address) (Connection, error)
	Dial(addr Address) (Connection, error)
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}
