package main

import (
	"bytes"
)

type Bucket struct {
	list []*Triple
}

// newBucket creates a new bucket with an initial capacity defined by BucketSize.
func newBucket() *Bucket {
	bucket := &Bucket{}
	bucket.list = make([]*Triple, 0, BucketSize)
	return bucket
}

// AddContact adds the Triple to the front of the bucket
// or moves it to the front of the bucket if it already existed
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
		// Contact doesn't exist, add it if there's space
		if len(b.list) < BucketSize {
			// Add to front
			b.list = append([]*Triple{&triple}, b.list...)
		}
		// If bucket is full, we don't add (following Kademlia spec)
		// In a full implementation, ping the least recently seen node
	} else {
		// Contact exists, move it to front
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

// GetFirst returns the first (least recently seen) Triple in the bucket
func (b *Bucket) GetFirst() *Triple {
	if len(b.list) == 0 {
		return nil
	}
	return b.list[len(b.list)-1] // Last element is least recently seen
}

// GetContactAndCalcDistance returns an array of Triples with calculated distances
func (b *Bucket) GetContactAndCalcDistance(targetID []byte) []Triple {
	var contacts []Triple

	for _, triple := range b.list {
		// Create a copy and calculate distance
		contact := *triple
		// You might want to add a Distance field to Triple or calculate it separately
		contacts = append(contacts, contact)
	}

	return contacts
}

// GetAllContacts returns all Triples in the bucket
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
