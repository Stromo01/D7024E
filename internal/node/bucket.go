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

func NewBucket() *Bucket {
	bucket := &Bucket{}
	bucket.List = make([]*Triple, 0, K)
	return bucket
}

func (b *Bucket) AddContact(triple Triple) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var index int = -1
	for i, t := range b.List { // Find if the contact already exists
		if bytes.Equal(t.ID, triple.ID) {
			index = i
			break
		}
	}

	if index == -1 { //Contact does not exist
		if b.IsFull() {
			// Remove least recently used contact (last element)
			b.List = b.List[:len(b.List)-1]
		}
		b.List = append([]*Triple{&triple}, b.List...) // Add new contact to front (most recent)
	} else { //Contact exists, move to front
		b.MoveToFront(triple)
	}
}

func (b *Bucket) RemoveContact(triple Triple) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, t := range b.List {
		if bytes.Equal(t.ID, triple.ID) {
			b.List = append(b.List[:i], b.List[i+1:]...) // Remove element at index i
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
	return len(b.List)
}

// IsFull checks if the bucket is at capacity
func (b *Bucket) IsFull() bool {
	return len(b.List) >= K
}

// GetLeastRecentlyUsed returns the contact that should be evicted
func (b *Bucket) GetLeastRecentlyUsed() *Triple {
	return b.GetLast()
}

// MoveToFront moves an existing contact to the front (most recent position)
func (b *Bucket) MoveToFront(triple Triple) bool {
	for i, t := range b.List {
		if bytes.Equal(t.ID, triple.ID) {
			contact := b.List[i]
			b.List = append(b.List[:i], b.List[i+1:]...)   // Remove from current position
			b.List = append([]*Triple{contact}, b.List...) // Add to front
			return true
		}
	}
	return false // Contact not found
}
