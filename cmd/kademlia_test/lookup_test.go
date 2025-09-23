package kademlia_test

import (
	"testing"

	"github.com/Stromo01/D7024E/cmd/kademlia"
)

func TestLookupBasic(t *testing.T) {
	// TODO: implement
}

func TestLookupStart(t *testing.T) {
	id := kademlia.NewRandomKademliaID()
	me := kademlia.NewContact(id, "127.0.0.1:8000")
	rt := kademlia.NewRoutingTable(me)
	net := &dummyNetwork{}
	lookup := kademlia.NewLookup(rt, net, id)
	lookup.Start()
}
