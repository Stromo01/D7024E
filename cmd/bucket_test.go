package main

import (
	"bytes"
	"testing"
)

func TestBucketInit(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	if bucket.list == nil {
		t.Error("Bucket list should be initialized")
	}

	if len(bucket.list) != 0 {
		t.Errorf("Expected empty bucket, got %d contacts", len(bucket.list))
	}

	if cap(bucket.list) != BucketSize {
		t.Errorf("Expected capacity %d, got %d", BucketSize, cap(bucket.list))
	}
}

func TestBucketAddContact(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	contact := &Triple{
		ID: []byte("test-id-1"),
		Addr: Address{
			IP:   "127.0.0.1",
			Port: 8001,
		},
	}

	// Add a contact
	bucket.addContact(contact)

	if len(bucket.list) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(bucket.list))
	}

	if !bytes.Equal(bucket.list[0].ID, contact.ID) {
		t.Error("Contact not added correctly")
	}
}

func TestBucketRemoveContact(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	contact1 := &Triple{
		ID:   []byte("test-id-1"),
		Addr: Address{IP: "127.0.0.1", Port: 8001},
	}

	contact2 := &Triple{
		ID:   []byte("test-id-2"),
		Addr: Address{IP: "127.0.0.1", Port: 8002},
	}

	// Add contacts
	bucket.addContact(contact1)
	bucket.addContact(contact2)

	if len(bucket.list) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(bucket.list))
	}

	// Remove the first contact
	bucket.removeContact(contact1.ID)

	if len(bucket.list) != 1 {
		t.Errorf("Expected 1 contact after removal, got %d", len(bucket.list))
	}

	if !bytes.Equal(bucket.list[0].ID, contact2.ID) {
		t.Error("Wrong contact was removed")
	}
}

func TestBucketContains(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	contact := &Triple{
		ID:   []byte("test-id-1"),
		Addr: Address{IP: "127.0.0.1", Port: 8001},
	}

	// Check before adding
	if bucket.contains(contact.ID) {
		t.Error("Empty bucket should not contain the contact")
	}

	// Add and check again
	bucket.addContact(contact)

	if !bucket.contains(contact.ID) {
		t.Error("Bucket should contain the added contact")
	}
}

func TestBucketGetContact(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	contact := &Triple{
		ID:   []byte("test-id-1"),
		Addr: Address{IP: "127.0.0.1", Port: 8001},
	}

	// Add and retrieve
	bucket.addContact(contact)

	retrieved := bucket.getContact(contact.ID)
	if retrieved == nil {
		t.Error("Failed to get contact")
	}

	if !bytes.Equal(retrieved.ID, contact.ID) {
		t.Error("Retrieved wrong contact")
	}

	// Try to get a non-existent contact
	nonexistent := bucket.getContact([]byte("non-existent"))
	if nonexistent != nil {
		t.Error("Should return nil for non-existent contact")
	}
}

func TestBucketFull(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	// Fill the bucket to capacity
	for i := 0; i < BucketSize; i++ {
		contact := &Triple{
			ID:   []byte(string(rune('a' + i))),
			Addr: Address{IP: "127.0.0.1", Port: 8000 + i},
		}
		bucket.addContact(contact)
	}

	if len(bucket.list) != BucketSize {
		t.Errorf("Expected bucket to have %d contacts, got %d", BucketSize, len(bucket.list))
	}

	// Try to add one more contact
	extraContact := &Triple{
		ID:   []byte("extra"),
		Addr: Address{IP: "127.0.0.1", Port: 9000},
	}
	bucket.addContact(extraContact)

	// The implementation should decide what to do when bucket is full
	// This test just verifies that no panic occurs and checks the bucket size
	if len(bucket.list) > BucketSize {
		t.Errorf("Bucket should not exceed maximum capacity of %d", BucketSize)
	}
}

func TestBucketLeastRecentlySeenContact(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	// Add a few contacts
	contact1 := &Triple{ID: []byte("id-1"), Addr: Address{IP: "127.0.0.1", Port: 8001}}
	contact2 := &Triple{ID: []byte("id-2"), Addr: Address{IP: "127.0.0.1", Port: 8002}}
	contact3 := &Triple{ID: []byte("id-3"), Addr: Address{IP: "127.0.0.1", Port: 8003}}

	bucket.addContact(contact1)
	bucket.addContact(contact2)
	bucket.addContact(contact3)

	// Assuming least recently seen is the first in the list
	// This depends on your specific implementation
	least := bucket.getLeastRecentlySeenContact()

	if least == nil {
		t.Error("Expected to get least recently seen contact")
		return
	}

	// This test might need adjustment based on your implementation
	if !bytes.Equal(least.ID, contact1.ID) {
		t.Errorf("Expected least recently seen to be first added contact")
	}
}

func TestBucketMoveToEnd(t *testing.T) {
	bucket := &Bucket{
		list: make([]*Triple, 0, BucketSize),
	}

	// Add contacts
	contact1 := &Triple{ID: []byte("id-1"), Addr: Address{IP: "127.0.0.1", Port: 8001}}
	contact2 := &Triple{ID: []byte("id-2"), Addr: Address{IP: "127.0.0.1", Port: 8002}}

	bucket.addContact(contact1)
	bucket.addContact(contact2)

	// Move the first contact to end (making it most recently seen)
	bucket.moveToEnd(contact1.ID)

	// Now the first contact should be the second one we added
	if !bytes.Equal(bucket.list[0].ID, contact2.ID) {
		t.Error("Failed to move contact to end of bucket")
	}
}
