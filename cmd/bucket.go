package main

import "bytes"

// BucketSize is the Kademlia bucket size
const BucketSize = 20

type Bucket struct {
	list []*Triple
}

// addContact adds a contact to the bucket if it doesn't already exist
// If the bucket is full, it removes the least recently seen contact
func (bucket *Bucket) addContact(contact *Triple) {
	// Check if contact already exists
	if bucket.contains(contact.ID) {
		// If it exists, move it to the end (most recently seen)
		bucket.moveToEnd(contact.ID)
		return
	}

	// If bucket is not full, add contact
	if len(bucket.list) < BucketSize {
		bucket.list = append(bucket.list, contact)
		return
	}

	// If bucket is full, use simplified replacement policy
	// Note: Proper Kademlia spec requires pinging the least recently seen contact
	// and only replacing if it doesn't respond. This is a simplified version.
	if len(bucket.list) > 0 {
		bucket.list = append(bucket.list[1:], contact)
	}
}

// removeContact removes a contact from the bucket by ID
func (bucket *Bucket) removeContact(id []byte) {
	for i, contact := range bucket.list {
		if bytes.Equal(contact.ID, id) {
			// Remove by slicing
			bucket.list = append(bucket.list[:i], bucket.list[i+1:]...)
			return
		}
	}
}

// contains checks if the bucket contains a contact with the given ID
func (bucket *Bucket) contains(id []byte) bool {
	return bucket.getContact(id) != nil
}

// getContact retrieves a contact by ID, or nil if not found
func (bucket *Bucket) getContact(id []byte) *Triple {
	for _, contact := range bucket.list {
		if bytes.Equal(contact.ID, id) {
			return contact
		}
	}
	return nil
}

// getLeastRecentlySeenContact returns the least recently seen contact (first in list)
func (bucket *Bucket) getLeastRecentlySeenContact() *Triple {
	if len(bucket.list) == 0 {
		return nil
	}
	return bucket.list[0]
}

// moveToEnd moves a contact to the end of the list (marking it as most recently seen)
func (bucket *Bucket) moveToEnd(id []byte) {
	for i, contact := range bucket.list {
		if bytes.Equal(contact.ID, id) {
			// Remove from current position
			bucket.list = append(bucket.list[:i], bucket.list[i+1:]...)
			// Add to the end
			bucket.list = append(bucket.list, contact)
			return
		}
	}
}

// size returns the number of contacts in the bucket
func (bucket *Bucket) size() int {
	return len(bucket.list)
}

// isFull returns true if the bucket is at capacity
func (bucket *Bucket) isFull() bool {
	return len(bucket.list) >= BucketSize
}

// getContacts returns a copy of all contacts in the bucket
func (bucket *Bucket) getContacts() []Triple {
	contacts := make([]Triple, len(bucket.list))
	for i, contact := range bucket.list {
		contacts[i] = *contact
	}
	return contacts
}
