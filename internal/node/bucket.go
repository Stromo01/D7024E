package node

import (
	"bytes"
	"sync"

	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

type Bucket struct {
	List []*Triple
	mu   sync.RWMutex
}

// NewBucket creates a new bucket with an initial capacity defined by BucketSize.
func NewBucket() *Bucket {
	bucket := &Bucket{}
	bucket.List = make([]*Triple, 0, K)
	return bucket
}

// AddContact adds the Triple to the front of the bucket (most recent)
func (b *Bucket) AddContact(triple Triple) {
	b.mu.Lock()
	defer b.mu.Unlock()

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
		if len(b.List) >= K {
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
	b.mu.Lock()
	defer b.mu.Unlock()

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
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, t := range b.List {
		if bytes.Equal(t.ID, triple.ID) {
			return true
		}
	}
	return false
}

// GetFirst returns the first (most recently seen) Triple in the bucket
func (b *Bucket) GetFirst() *Triple {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.List) == 0 {
		return nil
	}
	return b.List[0] // First element is most recently seen
}

// GetLast returns the last (least recently seen) Triple in the bucket
func (b *Bucket) GetLast() *Triple {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.List) == 0 {
		return nil
	}
	return b.List[len(b.List)-1] // Last element is least recently seen
}

// GetAllContacts returns all Triples in the bucket
func (b *Bucket) GetAllContacts() []Triple {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var contacts []Triple
	for _, triple := range b.List {
		contacts = append(contacts, *triple)
	}
	return contacts
}

// Len returns the size of the bucket
func (b *Bucket) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.List)
}

// IsFull checks if the bucket is at capacity
func (b *Bucket) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.List) >= K
}
