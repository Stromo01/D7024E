package kademlia_test

import (
	"testing"

	"github.com/Stromo01/D7024E/cmd/kademlia"
)

func TestContactBasic(t *testing.T) {
	// TODO: implement
}

func TestNewContactAndString(t *testing.T) {
	id := kademlia.NewRandomKademliaID()
	c := kademlia.NewContact(id, "127.0.0.1:8000")
	if c.ID == nil || c.Address != "127.0.0.1:8000" {
		t.Errorf("Contact not initialized correctly")
	}
	_ = c.String() // Just check it doesn't panic
}

func TestContactCalcDistanceAndLess(t *testing.T) {
	id1 := kademlia.NewRandomKademliaID()
	id2 := kademlia.NewRandomKademliaID()
	c1 := kademlia.NewContact(id1, "A")
	c2 := kademlia.NewContact(id2, "B")
	c1.CalcDistance(id2)
	c2.CalcDistance(id1)
	_ = c1.Less(&c2) // Just check it doesn't panic
}

func TestContactCandidatesSortAndGet(t *testing.T) {
	id1 := kademlia.NewRandomKademliaID()
	id2 := kademlia.NewRandomKademliaID()
	c1 := kademlia.NewContact(id1, "A")
	c2 := kademlia.NewContact(id2, "B")
	candidates := &kademlia.ContactCandidates{}
	candidates.Append([]kademlia.Contact{c1, c2})
	candidates.Sort()
	got := candidates.GetContacts(1)
	if len(got) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(got))
	}
}
