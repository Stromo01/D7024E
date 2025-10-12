package main

import (
	"bytes"
)

type Bucket struct {
	list []*Triple
}

// NewBucket creates a new bucket with an initial capacity defined by BucketSize.
func newBucket() *Bucket {
	bucket := &Bucket{}
	bucket.list = make([]*Triple, 0, BucketSize)
	return bucket
}

// AddContact adds the Triple to the front of the bucket (most recent)
// or moves it to the front if it already existed
// This maintains LRU order: [most recent ... least recent]
func (b *Bucket) AddContact(triple Triple) {
	var index int = -1

	// Find if the contact already exists
	for i, t := range b.list {
		if bytes.Equal(t.ID, triple.ID) {
			index = i
			break
		}
	}

	if index == -1 {
		// Check if bucket is full before adding new contact
		if b.IsFull() {
			// Remove least recently used contact (last element)
			b.list = b.list[:len(b.list)-1]
		}
		// Add new contact to front (most recent)
		b.list = append([]*Triple{&triple}, b.list...)
	} else {
		// Contact exists, move to front (most recent)
		contact := b.list[index]
		// Remove from current position
		b.list = append(b.list[:index], b.list[index+1:]...)
		// Add to front
		b.list = append([]*Triple{contact}, b.list...)
	}
}

// RemoveContact removes the Triple from the bucket
func (b *Bucket) RemoveContact(triple Triple) {
	for i, t := range b.list {
		if bytes.Equal(t.ID, triple.ID) {
			// Remove element at index i
			b.list = append(b.list[:i], b.list[i+1:]...)
			break
		}
	}
}

// Contains checks if the bucket contains the given Triple
func (b *Bucket) Contains(triple Triple) bool {
	for _, t := range b.list {
		if bytes.Equal(t.ID, triple.ID) {
			return true
		}
	}
	return false
}

// GetFirst returns the first (most recently seen) Triple in the bucket
func (b *Bucket) GetFirst() *Triple {
	if len(b.list) == 0 {
		return nil
	}
	return b.list[0] // First element is most recently seen
}

// GetLast returns the last (least recently seen) Triple in the bucket
func (b *Bucket) GetLast() *Triple {
	if len(b.list) == 0 {
		return nil
	}
	return b.list[len(b.list)-1] // Last element is least recently seen
}

// GetAllContacts returns all Triples in the bucket
// Ordered from most recently seen to least recently seen
func (b *Bucket) GetAllContacts() []Triple {
	var contacts []Triple
	for _, triple := range b.list {
		contacts = append(contacts, *triple)
	}
	return contacts
}

// Len returns the size of the bucket
func (b *Bucket) Len() int {
	return len(b.list)
}

// IsFull checks if the bucket is at capacity
func (b *Bucket) IsFull() bool {
	return len(b.list) >= BucketSize
}

// GetLeastRecentlyUsed returns the contact that should be evicted
// This is useful for bucket management when implementing ping-before-evict
func (b *Bucket) GetLeastRecentlyUsed() *Triple {
	return b.GetLast()
}

// MoveToFront moves an existing contact to the front (most recent position)
// This is useful when you receive a message from a known contact
func (b *Bucket) MoveToFront(triple Triple) bool {
	for i, t := range b.list {
		if bytes.Equal(t.ID, triple.ID) {
			// Remove from current position
			contact := b.list[i]
			b.list = append(b.list[:i], b.list[i+1:]...)
			// Add to front
			b.list = append([]*Triple{contact}, b.list...)
			return true
		}
	}
	return false // Contact not found
}
