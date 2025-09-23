package kademlia_test

import (
	"testing"

	"github.com/Stromo01/D7024E/cmd/kademlia"
)

func TestBucketAddContactAndLen(t *testing.T) {
	b := kademlia.NewBucket()
	id := kademlia.NewRandomKademliaID()
	c := kademlia.NewContact(id, "127.0.0.1:8000")
	b.AddContact(c, &dummyNetwork{})
	if b.Len() != 1 {
		t.Errorf("Expected bucket length 1, got %d", b.Len())
	}
}

func TestBucketGetContactAndCalcDistance(t *testing.T) {
	b := kademlia.NewBucket()
	id := kademlia.NewRandomKademliaID()
	c := kademlia.NewContact(id, "127.0.0.1:8000")
	b.AddContact(c, &dummyNetwork{})
	contacts := b.GetContactAndCalcDistance(id)
	if len(contacts) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(contacts))
	}
}
