package kademlia

import "github.com/Stromo01/D7024E/cmd/kademlia"

// dummyNetwork implements kademlia.Network for testing
// Place in a single file to avoid redeclaration errors in multiple test files.
type dummyNetwork struct{}

func (d *dummyNetwork) Listen(addr kademlia.Address) (kademlia.Connection, error) { return nil, nil }
func (d *dummyNetwork) Dial(addr kademlia.Address) (kademlia.Connection, error)   { return nil, nil }
