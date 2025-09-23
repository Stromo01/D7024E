package kademlia_test

import (
	"testing"

	"github.com/Stromo01/D7024E/cmd/kademlia"
)

func TestRoutingTableBasic(t *testing.T) {
	// TODO: implement
}

func TestRoutingTableAddAndFindClosest(t *testing.T) {
	id := kademlia.NewRandomKademliaID()
	me := kademlia.NewContact(id, "127.0.0.1:8000")
	rt := kademlia.NewRoutingTable(me)
	c := kademlia.NewContact(kademlia.NewRandomKademliaID(), "127.0.0.1:8001")
	rt.AddContact(c, &dummyNetwork{})
	closest := rt.FindClosestContacts(id, 1)
	if len(closest) == 0 {
		t.Errorf("Expected at least one contact in closest")
	}
}
