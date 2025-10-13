package node

import (
	"bytes"

	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

type Bucket struct {
	List []*Triple
}

// NewBucket creates a new bucket with an initial capacity defined by BucketSize.
func NewBucket() *Bucket {
	bucket := &Bucket{}
	bucket.List = make([]*Triple, 0, BucketSize)
	return bucket
}

// AddContact adds the Triple to the front of the bucket (most recent)
// or moves it to the front if it already existed
// This maintains LRU order: [most recent ... least recent]
func (b *Bucket) AddContact(triple Triple) {
	var index int = -1

	// Find if the contact already exists
	for i, t := range b.List {
		if bytes.Equal(t.ID, triple.ID) {
			index = i
			break
		}
	}

	if index == -1 {
		// Check if bucket is full before adding new contact
		if b.IsFull() {
			// Remove least recently used contact (last element)
			b.List = b.List[:len(b.List)-1]
		}
		// Add new contact to front (most recent)
		b.List = append([]*Triple{&triple}, b.List...)
	} else {
		// Contact exists, move to front (most recent)
		contact := b.List[index]
		// Remove from current position
		b.List = append(b.List[:index], b.List[index+1:]...)
		// Add to front
		b.List = append([]*Triple{contact}, b.List...)
	}
}

// RemoveContact removes the Triple from the bucket
func (b *Bucket) RemoveContact(triple Triple) {
	for i, t := range b.List {
		if bytes.Equal(t.ID, triple.ID) {
			// Remove element at index i
			b.List = append(b.List[:i], b.List[i+1:]...)
			break
		}
	}
}

// Contains checks if the bucket contains the given Triple
func (b *Bucket) Contains(triple Triple) bool {
	for _, t := range b.List {
		if bytes.Equal(t.ID, triple.ID) {
			return true
		}
	}
	return false
}

// GetFirst returns the first (most recently seen) Triple in the bucket
func (b *Bucket) GetFirst() *Triple {
	if len(b.List) == 0 {
		return nil
	}
	return b.List[0] // First element is most recently seen
}

// GetLast returns the last (least recently seen) Triple in the bucket
func (b *Bucket) GetLast() *Triple {
	if len(b.List) == 0 {
		return nil
	}
	return b.List[len(b.List)-1] // Last element is least recently seen
}

// GetAllContacts returns all Triples in the bucket
// Ordered from most recently seen to least recently seen
func (b *Bucket) GetAllContacts() []Triple {
	var contacts []Triple
	for _, triple := range b.List {
		contacts = append(contacts, *triple)
	}
	return contacts
}

// Len returns the size of the bucket
func (b *Bucket) Len() int {
	return len(b.List)
}

// IsFull checks if the bucket is at capacity
func (b *Bucket) IsFull() bool {
	return len(b.List) >= BucketSize
}

// GetLeastRecentlyUsed returns the contact that should be evicted
// This is useful for bucket management when implementing ping-before-evict
func (b *Bucket) GetLeastRecentlyUsed() *Triple {
	return b.GetLast()
}

// MoveToFront moves an existing contact to the front (most recent position)
// This is useful when you receive a message from a known contact
func (b *Bucket) MoveToFront(triple Triple) bool {
	for i, t := range b.List {
		if bytes.Equal(t.ID, triple.ID) {
			// Remove from current position
			contact := b.List[i]
			b.List = append(b.List[:i], b.List[i+1:]...)
			// Add to front
			b.List = append([]*Triple{contact}, b.List...)
			return true
		}
	}
	return false // Contact not found
}
