package node_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	. "github.com/eislab-cps/go-template/internal/node"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// Helper function to create a random Triple for testing
func createRandomTriple() Triple {
	var id [20]byte
	rand.Read(id[:])
	return Triple{
		ID:   id[:],
		Addr: Address{IP: "127.0.0.1", Port: 8000},
		Port: 8000,
	}
}

func TestBucketLRUOrdering(t *testing.T) {
	bucket := NewBucket()

	// Add contacts
	contact1 := createTripleWithID([]byte{1})
	contact2 := createTripleWithID([]byte{2})
	contact3 := createTripleWithID([]byte{3})

	bucket.AddContact(contact1)
	bucket.AddContact(contact2)
	bucket.AddContact(contact3)

	// contact3 should be at front (most recent)
	first := bucket.GetFirst()
	if !bytes.Equal(first.ID, contact3.ID) {
		t.Errorf("Expected most recent contact to be contact3")
	}

	// contact1 should be at back (least recent)
	last := bucket.GetLast()
	if !bytes.Equal(last.ID, contact1.ID) {
		t.Errorf("Expected least recent contact to be contact1")
	}

	// Move contact1 to front by adding it again
	bucket.AddContact(contact1)

	// Now contact1 should be most recent
	first = bucket.GetFirst()
	if !bytes.Equal(first.ID, contact1.ID) {
		t.Errorf("Expected contact1 to move to front")
	}
}

// Helper function to create a Triple with specific ID
func createTripleWithID(id []byte) Triple {
	return Triple{
		ID:   id,
		Addr: Address{IP: "127.0.0.1", Port: 8000},
		Port: 8000,
	}
}

func TestNewBucket(t *testing.T) {
	bucket := NewBucket()

	if bucket == nil {
		t.Fatal("NewBucket() returned nil")
	}

	if bucket.Len() != 0 {
		t.Errorf("Expected empty bucket, got length %d", bucket.Len())
	}

	if bucket.IsFull() {
		t.Error("New bucket should not be full")
	}
}

func TestBucketAddContact(t *testing.T) {
	bucket := NewBucket()
	triple := createRandomTriple()

	// Test adding new contact
	bucket.AddContact(triple)

	if bucket.Len() != 1 {
		t.Errorf("Expected bucket length 1, got %d", bucket.Len())
	}

	if !bucket.Contains(triple) {
		t.Error("Bucket should contain the added contact")
	}

	// Test adding same contact again (should move to front)
	bucket.AddContact(triple)

	if bucket.Len() != 1 {
		t.Errorf("Expected bucket length 1 after re-adding same contact, got %d", bucket.Len())
	}

	// Verify it's at the front (index 0)
	if !bytes.Equal(bucket.List[0].ID, triple.ID) {
		t.Error("Re-added contact should be moved to front")
	}
}

func TestBucketAddContactOrder(t *testing.T) {
	bucket := NewBucket()
	triple1 := createRandomTriple()
	triple2 := createRandomTriple()

	// Add first contact
	bucket.AddContact(triple1)
	// Add second contact
	bucket.AddContact(triple2)

	// triple2 should be at front (index 0), triple1 at back (index 1)
	if !bytes.Equal(bucket.List[0].ID, triple2.ID) {
		t.Error("Most recently added contact should be at front")
	}

	if !bytes.Equal(bucket.List[1].ID, triple1.ID) {
		t.Error("Less recently added contact should be at back")
	}
}

func TestBucketAddContactToFull(t *testing.T) {
	bucket := NewBucket()

	// Fill the bucket to capacity
	contacts := make([]Triple, K)
	for i := 0; i < K; i++ {
		contacts[i] = createRandomTriple()
		bucket.AddContact(contacts[i])
	}

	if !bucket.IsFull() {
		t.Error("Bucket should be full")
	}

	// Remember the LRU contact (should be contacts[0] at the back)
	lruContact := bucket.GetLast()

	// Try to add one more contact - should evict LRU and add new contact
	newContact := createRandomTriple()
	bucket.AddContact(newContact)

	if bucket.Len() != K {
		t.Errorf("Bucket length should remain %d after adding to full bucket, got %d", K, bucket.Len())
	}

	if !bucket.Contains(newContact) {
		t.Error("New contact should be added to full bucket (with LRU eviction)")
	}

	if bucket.Contains(*lruContact) {
		t.Error("LRU contact should have been evicted")
	}

	// New contact should be at the front
	first := bucket.GetFirst()
	if !bytes.Equal(first.ID, newContact.ID) {
		t.Error("New contact should be at front (most recent)")
	}
}

func TestBucketAddExistingContactToFull(t *testing.T) {
	bucket := NewBucket()

	// Fill the bucket to capacity
	contacts := make([]Triple, K)
	for i := 0; i < K; i++ {
		contacts[i] = createRandomTriple()
		bucket.AddContact(contacts[i])
	}

	// Try to re-add the first contact (should move to front)
	firstContact := contacts[0]
	bucket.AddContact(firstContact)

	if bucket.Len() != K {
		t.Errorf("Bucket length should remain %d when re-adding existing contact, got %d", K, bucket.Len())
	}

	// Should be moved to front
	if !bytes.Equal(bucket.List[0].ID, firstContact.ID) {
		t.Error("Re-added existing contact should be moved to front")
	}
}

func TestBucketRemoveContact(t *testing.T) {
	bucket := NewBucket()
	triple1 := createRandomTriple()
	triple2 := createRandomTriple()

	bucket.AddContact(triple1)
	bucket.AddContact(triple2)

	if bucket.Len() != 2 {
		t.Errorf("Expected bucket length 2, got %d", bucket.Len())
	}

	// Remove first contact
	bucket.RemoveContact(triple1)

	if bucket.Len() != 1 {
		t.Errorf("Expected bucket length 1 after removal, got %d", bucket.Len())
	}

	if bucket.Contains(triple1) {
		t.Error("Bucket should not contain removed contact")
	}

	if !bucket.Contains(triple2) {
		t.Error("Bucket should still contain non-removed contact")
	}
}

func TestBucketRemoveNonExistentContact(t *testing.T) {
	bucket := NewBucket()
	triple1 := createRandomTriple()
	triple2 := createRandomTriple()

	bucket.AddContact(triple1)

	// Try to remove contact that doesn't exist
	bucket.RemoveContact(triple2)

	if bucket.Len() != 1 {
		t.Errorf("Expected bucket length 1 after removing non-existent contact, got %d", bucket.Len())
	}

	if !bucket.Contains(triple1) {
		t.Error("Bucket should still contain original contact")
	}
}

func TestBucketRemoveFromEmpty(t *testing.T) {
	bucket := NewBucket()
	triple := createRandomTriple()

	// Try to remove from empty bucket
	bucket.RemoveContact(triple)

	if bucket.Len() != 0 {
		t.Errorf("Expected bucket length 0, got %d", bucket.Len())
	}
}

func TestBucketContains(t *testing.T) {
	bucket := NewBucket()
	triple1 := createRandomTriple()
	triple2 := createRandomTriple()

	// Test empty bucket
	if bucket.Contains(triple1) {
		t.Error("Empty bucket should not contain any contact")
	}

	// Add contact and test
	bucket.AddContact(triple1)

	if !bucket.Contains(triple1) {
		t.Error("Bucket should contain added contact")
	}

	if bucket.Contains(triple2) {
		t.Error("Bucket should not contain non-added contact")
	}
}

func TestBucketGetFirst(t *testing.T) {
	bucket := NewBucket()

	// Test empty bucket
	first := bucket.GetFirst()
	if first != nil {
		t.Error("GetFirst() should return nil for empty bucket")
	}

	// Add some contacts
	triple1 := createRandomTriple()
	triple2 := createRandomTriple()

	bucket.AddContact(triple1)
	bucket.AddContact(triple2)

	// triple2 should be the most recently added (at front)
	first = bucket.GetFirst()
	if first == nil {
		t.Fatal("GetFirst() should not return nil for non-empty bucket")
	}

	if !bytes.Equal(first.ID, triple2.ID) {
		t.Error("GetFirst() should return the most recently seen contact")
	}
}

func TestBucketGetFirstSingleContact(t *testing.T) {
	bucket := NewBucket()
	triple := createRandomTriple()

	bucket.AddContact(triple)

	first := bucket.GetFirst()
	if first == nil {
		t.Fatal("GetFirst() should not return nil for bucket with one contact")
	}

	if !bytes.Equal(first.ID, triple.ID) {
		t.Error("GetFirst() should return the only contact")
	}
}

func TestBucketGetAllContacts(t *testing.T) {
	bucket := NewBucket()
	triple1 := createRandomTriple()
	triple2 := createRandomTriple()

	bucket.AddContact(triple1)
	bucket.AddContact(triple2)

	contacts := bucket.GetAllContacts()

	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}

	// Verify the contacts are copies
	originalPort := bucket.List[0].Port
	contacts[0].Port = 9999
	if bucket.List[0].Port != originalPort {
		t.Error("GetAllContacts should return copies, not references")
	}

	// Verify order is preserved (most recent first)
	if !bytes.Equal(contacts[0].ID, triple2.ID) {
		t.Error("First contact should be the most recently added")
	}

	if !bytes.Equal(contacts[1].ID, triple1.ID) {
		t.Error("Second contact should be the first added")
	}
}

func TestBucketGetAllContactsEmpty(t *testing.T) {
	bucket := NewBucket()

	contacts := bucket.GetAllContacts()

	if len(contacts) != 0 {
		t.Errorf("Expected 0 contacts for empty bucket, got %d", len(contacts))
	}
}

func TestBucketLen(t *testing.T) {
	bucket := NewBucket()

	// Test empty bucket
	if bucket.Len() != 0 {
		t.Errorf("Expected length 0 for empty bucket, got %d", bucket.Len())
	}

	// Add contacts and test length
	for i := 1; i <= K-1; i++ {
		triple := createRandomTriple()
		bucket.AddContact(triple)

		if bucket.Len() != i {
			t.Errorf("Expected length %d after adding %d contacts, got %d", i, i, bucket.Len())
		}
	}
}

func TestBucketIsFull(t *testing.T) {
	bucket := NewBucket()

	// Test empty bucket
	if bucket.IsFull() {
		t.Error("Empty bucket should not be full")
	}

	// Fill bucket to capacity - 1
	for i := 0; i < K-1; i++ {
		triple := createRandomTriple()
		bucket.AddContact(triple)
	}

	if bucket.IsFull() {
		t.Error("Bucket should not be full when below capacity")
	}

	// Add one more to reach capacity
	triple := createRandomTriple()
	bucket.AddContact(triple)

	if !bucket.IsFull() {
		t.Error("Bucket should be full when at capacity")
	}
}

func TestBucketLRUBehavior(t *testing.T) {
	bucket := NewBucket()

	// Add 3 contacts
	triple1 := createRandomTriple()
	triple2 := createRandomTriple()
	triple3 := createRandomTriple()

	bucket.AddContact(triple1) // Oldest
	bucket.AddContact(triple2)
	bucket.AddContact(triple3) // Newest

	// triple3 should be at front, triple1 at back
	if !bytes.Equal(bucket.List[0].ID, triple3.ID) {
		t.Error("Most recently added should be at front")
	}

	if !bytes.Equal(bucket.List[2].ID, triple1.ID) {
		t.Error("Least recently added should be at back")
	}

	// Re-add triple1 (should move to front)
	bucket.AddContact(triple1)

	if !bytes.Equal(bucket.List[0].ID, triple1.ID) {
		t.Error("Re-added contact should be moved to front")
	}

	// triple2 should now be the least recently seen
	leastRecent := bucket.GetLast() // Use GetLast, not GetFirst!
	if !bytes.Equal(leastRecent.ID, triple2.ID) {
		t.Error("GetLast should return the least recently seen contact")
	}
}
