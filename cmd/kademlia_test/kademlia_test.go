package kademlia_test

import (
	"testing"

	"github.com/Stromo01/D7024E/cmd/kademlia"
)

func TestKademliaNodeBasic(t *testing.T) {
	// TODO: implement
}

func TestKademliaNewAndLookupContact(t *testing.T) {
	id := kademlia.NewRandomKademliaID()
	me := kademlia.NewContact(id, "127.0.0.1:8000")
	rt := kademlia.NewRoutingTable(me)
	// Use a dummy network implementation for now
	net := &dummyNetwork{}
	kad := kademlia.NewKademlia(rt, net)
	if kad.RoutingTable != rt || kad.Network != net {
		t.Errorf("Kademlia struct not initialized correctly")
	}
	// LookupContact should not panic
	kad.LookupContact(id)
}
