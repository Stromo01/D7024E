package node_test

import (
	"bytes"
	"math/big"
	"testing"

	. "github.com/eislab-cps/go-template/internal/node"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

func TestNewRoutingTable(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	if rt == nil {
		t.Fatal("NewRoutingTable() returned nil")
	}

	if !bytes.Equal(rt.Me.ID, Me.ID) {
		t.Error("Routing table should store the correct node ID")
	}

	// Check that all Buckets are initialized
	expectedBuckets := IDLength * 8
	if len(rt.Buckets) != expectedBuckets {
		t.Errorf("Expected %d Buckets, got %d", expectedBuckets, len(rt.Buckets))
	}

	for i, bucket := range rt.Buckets {
		if bucket == nil {
			t.Errorf("Bucket %d should be initialized", i)
		}
		if bucket.Len() != 0 {
			t.Errorf("Bucket %d should be empty initially", i)
		}
	}
}

func TestRoutingTableAddContact(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)
	contact := createRandomTriple()

	rt.AddContact(contact)

	// Since we can't access GetBucketIndex directly, test through GetKClosest
	closest := rt.GetKClosest("test", 10)

	found := false
	for _, c := range closest {
		if bytes.Equal(c.ID, contact.ID) {
			found = true
			break
		}
	}

	if !found {
		t.Error("Contact should be added to routing table")
	}
}

func TestRoutingTableAddMultipleContacts(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	contacts := make([]Triple, 10)
	for i := 0; i < 10; i++ {
		contacts[i] = createRandomTriple()
		rt.AddContact(contacts[i])
	}

	// Verify all contacts are added
	totalContacts := 0
	for _, bucket := range rt.Buckets {
		totalContacts += bucket.Len()
	}

	if totalContacts != 10 {
		t.Errorf("Expected 10 total contacts, got %d", totalContacts)
	}

	// Verify each contact is in the correct bucket
	for _, contact := range contacts {
		bucketIndex := rt.GetBucketIndex(contact.ID)
		bucket := rt.Buckets[bucketIndex]

		if !bucket.Contains(contact) {
			t.Errorf("Contact should be in bucket %d", bucketIndex)
		}
	}
}

func TestRoutingTableRemoveContact(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)
	contact := createRandomTriple()

	// Add contact first
	rt.AddContact(contact)

	// Verify it's added
	bucketIndex := rt.GetBucketIndex(contact.ID)
	bucket := rt.Buckets[bucketIndex]
	if !bucket.Contains(contact) {
		t.Fatal("Contact should be added before removal test")
	}

	// Remove contact
	rt.RemoveContact(contact)

	if bucket.Contains(contact) {
		t.Error("Contact should be removed from the routing table")
	}

	if bucket.Len() != 0 {
		t.Error("Bucket should be empty after removing the only contact")
	}
}

func TestRoutingTableRemoveNonExistentContact(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)
	contact1 := createRandomTriple()
	contact2 := createRandomTriple()

	rt.AddContact(contact1)

	// Try to remove contact that doesn't exist
	rt.RemoveContact(contact2)

	// contact1 should still be there
	bucketIndex := rt.GetBucketIndex(contact1.ID)
	bucket := rt.Buckets[bucketIndex]

	if !bucket.Contains(contact1) {
		t.Error("Original contact should still be in routing table")
	}
}

func TestRoutingTableGetBucketIndex(t *testing.T) {
	Me := createTripleWithID(make([]byte, 20)) // All zeros
	rt := NewRoutingTable(Me)

	testCases := []struct {
		description string
		id          []byte
		expectedMin int
		expectedMax int
	}{
		{
			description: "identical ID",
			id:          make([]byte, 20),
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			description: "distance 1",
			id: func() []byte {
				id := make([]byte, 20)
				id[19] = 1
				return id
			}(),
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			description: "distance 2",
			id: func() []byte {
				id := make([]byte, 20)
				id[19] = 2
				return id
			}(),
			expectedMin: 0,
			expectedMax: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			index := rt.GetBucketIndex(tc.id)

			if index < tc.expectedMin || index > tc.expectedMax {
				t.Errorf("Expected bucket index between %d and %d for %s, got %d",
					tc.expectedMin, tc.expectedMax, tc.description, index)
			}

			if index < 0 || index >= len(rt.Buckets) {
				t.Errorf("Bucket index %d is out of range [0, %d)", index, len(rt.Buckets))
			}
		})
	}
}

func TestRoutingTableGetBucketIndexBounds(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	// Test with various random IDs to ensure bucket index is always valid
	for i := 0; i < 100; i++ {
		contact := createRandomTriple()
		index := rt.GetBucketIndex(contact.ID)

		if index < 0 || index >= len(rt.Buckets) {
			t.Errorf("Bucket index %d is out of range [0, %d) for iteration %d",
				index, len(rt.Buckets), i)
		}
	}
}

func TestRoutingTableGetKClosest(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	// Add some contacts
	contacts := make([]Triple, 10)
	for i := 0; i < 10; i++ {
		contacts[i] = createRandomTriple()
		rt.AddContact(contacts[i])
	}

	targetKey := "test_key"
	closest := rt.GetKClosest(targetKey, 5)

	if len(closest) > 5 {
		t.Errorf("Expected at most 5 closest contacts, got %d", len(closest))
	}

	if len(closest) > 10 {
		t.Error("Cannot return more contacts than exist in routing table")
	}

	// Verify the results are sorted by distance
	if len(closest) > 1 {
		keyBytes := []byte(targetKey)
		for i := 1; i < len(closest); i++ {
			dist1 := XorDistance(keyBytes, closest[i-1].ID)
			dist2 := XorDistance(keyBytes, closest[i].ID)
			if dist1.Cmp(dist2) > 0 {
				t.Error("Results should be sorted by distance (closest first)")
			}
		}
	}
}

func TestRoutingTableGetKClosestEmptyTable(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	closest := rt.GetKClosest("test_key", 5)

	if len(closest) != 0 {
		t.Errorf("Expected empty result for empty routing table, got %d contacts", len(closest))
	}
}

func TestRoutingTableGetKClosestMoreThanAvailable(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	// Add only 3 contacts
	for i := 0; i < 3; i++ {
		contact := createRandomTriple()
		rt.AddContact(contact)
	}

	// Request 10 closest (more than available)
	closest := rt.GetKClosest("test_key", 10)

	if len(closest) != 3 {
		t.Errorf("Expected 3 contacts (all available), got %d", len(closest))
	}
}

func TestRoutingTableGetKClosestSingleContact(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	contact := createRandomTriple()
	rt.AddContact(contact)

	closest := rt.GetKClosest("test_key", 5)

	if len(closest) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(closest))
	}

	if !bytes.Equal(closest[0].ID, contact.ID) {
		t.Error("Should return the only contact in the table")
	}
}

func TestRoutingTableGetKClosestZeroK(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	contact := createRandomTriple()
	rt.AddContact(contact)

	closest := rt.GetKClosest("test_key", 0)

	if len(closest) != 0 {
		t.Errorf("Expected 0 contacts when K=0, got %d", len(closest))
	}
}

func TestRoutingTableBucketDistribution(t *testing.T) {
	Me := createTripleWithID(make([]byte, 20)) // All zeros for predictable bucket distribution
	rt := NewRoutingTable(Me)

	// Create contacts with known bit patterns to test distribution
	testContacts := []struct {
		description string
		id          []byte
		shouldAdd   bool // Whether we expect this contact to be added
	}{
		{
			description: "identical ID",
			id:          make([]byte, 20),
			shouldAdd:   false, // Should not be added (it's ourselves)
		},
		{
			description: "differ in last bit",
			id: func() []byte {
				id := make([]byte, 20)
				id[19] = 0x01 // ...00000001
				return id
			}(),
			shouldAdd: true,
		},
		{
			description: "differ in second last bit",
			id: func() []byte {
				id := make([]byte, 20)
				id[19] = 0x02 // ...00000010
				return id
			}(),
			shouldAdd: true,
		},
		{
			description: "differ in multiple bits",
			id: func() []byte {
				id := make([]byte, 20)
				id[19] = 0xFF // ...11111111
				return id
			}(),
			shouldAdd: true,
		},
	}

	for _, tc := range testContacts {
		t.Run(tc.description, func(t *testing.T) {
			contact := createTripleWithID(tc.id)
			bucketIndex := rt.GetBucketIndex(contact.ID)

			// Add the contact
			rt.AddContact(contact)

			// Verify it's in the correct bucket (or not added if it's identical)
			bucket := rt.Buckets[bucketIndex]

			if tc.shouldAdd {
				if !bucket.Contains(contact) {
					t.Errorf("Contact with %s should be in bucket %d", tc.description, bucketIndex)
				}
			} else {
				if bucket.Contains(contact) {
					t.Errorf("Contact with %s should not be added (identical to self)", tc.description)
				}
			}
		})
	}
}

func TestRoutingTableBucketCapacity(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	// Try to add more contacts to a single bucket than its capacity
	// We'll use contacts with IDs that hash to the same bucket

	// Add contacts until bucket is full
	contacts := make([]Triple, BucketSize+5)
	for i := 0; i < BucketSize+5; i++ {
		// Create contacts that will go to the same bucket by using similar IDs
		contacts[i] = createRandomTriple()

		// Ensure they go to the same bucket by manipulating the distance
		// This is a simplification - in practice, you'd need more sophisticated ID generation
		rt.AddContact(contacts[i])
	}

	// Check that no bucket exceeds capacity
	for i, bucket := range rt.Buckets {
		if bucket.Len() > BucketSize {
			t.Errorf("Bucket %d has %d contacts, should not exceed %d", i, bucket.Len(), BucketSize)
		}
	}
}

func TestRoutingTableIntegrationWithBucket(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	// Add, remove, and re-add contacts to test integration
	contact1 := createRandomTriple()
	contact2 := createRandomTriple()

	// Add contacts
	rt.AddContact(contact1)
	rt.AddContact(contact2)

	// Get K closest and verify they're included
	closest := rt.GetKClosest("test_key", 10)

	found1, found2 := false, false
	for _, c := range closest {
		if bytes.Equal(c.ID, contact1.ID) {
			found1 = true
		}
		if bytes.Equal(c.ID, contact2.ID) {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Both added contacts should be found in K closest results")
	}

	// Remove one contact
	rt.RemoveContact(contact1)

	// Verify it's no longer in K closest
	closest = rt.GetKClosest("test_key", 10)

	found1 = false
	for _, c := range closest {
		if bytes.Equal(c.ID, contact1.ID) {
			found1 = true
		}
	}

	if found1 {
		t.Error("Removed contact should not be in K closest results")
	}

	// Re-add and verify it's back
	rt.AddContact(contact1)
	closest = rt.GetKClosest("test_key", 10)

	found1 = false
	for _, c := range closest {
		if bytes.Equal(c.ID, contact1.ID) {
			found1 = true
		}
	}

	if !found1 {
		t.Error("Re-added contact should be in K closest results")
	}
}

func TestXorDistanceFunctionForRouting(t *testing.T) {
	// Test XOR distance function that routing table depends on
	a := []byte{0x00, 0x00}
	b := []byte{0x00, 0x01}

	distance := XorDistance(a, b)
	expected := big.NewInt(1)

	if distance.Cmp(expected) != 0 {
		t.Errorf("Expected distance %s, got %s", expected.String(), distance.String())
	}

	// Test XOR distance is symmetric
	distance1 := XorDistance(a, b)
	distance2 := XorDistance(b, a)

	if distance1.Cmp(distance2) != 0 {
		t.Error("XOR distance should be symmetric")
	}

	// Test XOR distance to self is 0
	distance = XorDistance(a, a)
	if distance.Sign() != 0 {
		t.Error("XOR distance to self should be 0")
	}
}

func TestRoutingTableStressTest(t *testing.T) {
	Me := createRandomTriple()
	rt := NewRoutingTable(Me)

	// Add many contacts and verify table remains consistent
	numContacts := 1000
	contacts := make([]Triple, numContacts)

	for i := 0; i < numContacts; i++ {
		contacts[i] = createRandomTriple()
		rt.AddContact(contacts[i])
	}

	// Verify we can still get K closest without errors
	closest := rt.GetKClosest("stress_test_key", K)

	if len(closest) == 0 && numContacts > 0 {
		t.Error("Should be able to find contacts in stress test")
	}

	// Verify distance ordering is maintained
	if len(closest) > 1 {
		keyBytes := []byte("stress_test_key")
		for i := 1; i < len(closest); i++ {
			dist1 := XorDistance(keyBytes, closest[i-1].ID)
			dist2 := XorDistance(keyBytes, closest[i].ID)
			if dist1.Cmp(dist2) > 0 {
				t.Error("Distance ordering not maintained in stress test")
			}
		}
	}

	// Count total contacts stored
	totalStored := 0
	for _, bucket := range rt.Buckets {
		totalStored += bucket.Len()
	}

	t.Logf("Stored %d out of %d contacts across %d Buckets", totalStored, numContacts, len(rt.Buckets))

	// Should have stored some contacts (exact number depends on distribution)
	if totalStored == 0 {
		t.Error("Should have stored some contacts")
	}
}
