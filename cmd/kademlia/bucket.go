
// bucket.go: Kademlia k-bucket implementation placeholder

package kademlia

import "container/list"

// NewBucket returns a new k-bucket (exported for testing)
func NewBucket() *bucket {
	return newBucket()
}

type bucket struct {
	list *list.List
}

func newBucket() *bucket {
	b := &bucket{}
	b.list = list.New()
	return b
}

func (b *bucket) AddContact(contact Contact, net Network) {
	var element *list.Element
	for e := b.list.Front(); e != nil; e = e.Next() {
		if contact.ID.Equals(e.Value.(Contact).ID) {
			element = e
		}
	}
	if element != nil {
		b.list.MoveToFront(element)
	} else {
		if b.list.Len() < BucketSize {
			b.list.PushFront(contact)
		} else {
			// TODO: implement ping/eviction logic
			b.list.Remove(b.list.Back())
			b.list.PushFront(contact)
		}
	}
}

func (b *bucket) GetContactAndCalcDistance(target *KademliaID) []Contact {
	var contacts []Contact
	for elt := b.list.Front(); elt != nil; elt = elt.Next() {
		contact := elt.Value.(Contact)
		contact.CalcDistance(target)
		contacts = append(contacts, contact)
	}
	return contacts
}

func (b *bucket) Len() int {
	return b.list.Len()
}
