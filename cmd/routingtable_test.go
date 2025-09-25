package main

import (
	"bytes"
	"math/big"
	"testing"
)

func TestRoutingTableInit(t *testing.T) {
	me := Triple{
		ID: []byte("0000000000000000000000"),
		Addr: Address{
			IP:   "127.0.0.1",
			Port: 8000,
		},
	}

	rt := NewRoutingTable(me)

	// Check if all buckets are initialized
	for i, bucket := range rt.buckets {
		if bucket == nil {
			t.Errorf("Bucket at index %d is nil", i)
		}
		if len(bucket.list) != 0 {
			t.Errorf("Expected empty bucket at index %d, got %d contacts", i, len(bucket.list))
		}
	}

	// Verify the node's own ID is stored
	if !bytes.Equal(rt.me.ID, me.ID) {
		t.Errorf("Expected routing table to store the node's own ID")
	}
}

// Helper function to create a test Triple with a given ID
func createTestTriple(id string, port int) Triple {
	return Triple{
		ID: []byte(id),
		Addr: Address{
			IP:   "127.0.0.1",
			Port: port,
		},
	}
}

// Create contact with a specific distance from the reference node
func createTripleWithDistance(referenceID []byte, distance *big.Int, port int) Triple {
	// Use XOR distance to calculate an ID with the specified distance from referenceID
	// This is a simplified implementation for testing
	distanceBytes := distance.Bytes()
	newID := make([]byte, len(referenceID))

	// Apply XOR to get a new ID with the desired distance
	for i := 0; i < len(referenceID) && i < len(distanceBytes); i++ {
		newID[i] = referenceID[i] ^ distanceBytes[len(distanceBytes)-i-1]
	}

	return Triple{
		ID: newID,
		Addr: Address{
			IP:   "127.0.0.1",
			Port: port,
		},
	}
}

func TestAddContact(t *testing.T) {
	meID := []byte("0000000000000000000000")
	me := Triple{
		ID: meID,
		Addr: Address{
			IP:   "127.0.0.1",
			Port: 8000,
		},
	}

	rt := NewRoutingTable(me)

	// Create and add a contact
	contact := createTestTriple("1111111111111111111111", 8001)
	rt.addContact(contact)

	// Test that a contact is added correctly
	distance := xorDistance(rt.me.ID, contact.ID)
	bucketIndex := getBucketIndex(distance)

	found := false
	for _, c := range rt.buckets[bucketIndex].list {
		if bytes.Equal(c.ID, contact.ID) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Contact was not added to the correct bucket")
	}
}

// Helper function to calculate bucket index from distance
func getBucketIndex(distance *big.Int) int {
	// This should match the logic in getBucketIndexFromDistance
	if distance.Sign() == 0 {
		return 0 // If the distance is 0, use bucket 0
	}

	// Make sure we don't exceed the bucket array size
	bucketIndex := distance.BitLen() - 1
	if bucketIndex >= IDLength*8 {
		bucketIndex = IDLength*8 - 1
	}
	return bucketIndex
}

func TestXorDistance(t *testing.T) {
	// Test known distances - use actual binary data instead of strings
	id1 := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}   // all zeros
	id2 := []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // first bit set (0x80)

	distance := xorDistance(id1, id2)

	// The first byte should be 0x80 (128 decimal)
	expectedFirstByte := byte(0x80) // binary 10000000
	if len(distance.Bytes()) == 0 || distance.Bytes()[0] != expectedFirstByte {
		t.Errorf("Expected distance first byte to be %x, got %v", expectedFirstByte, distance.Bytes())
	}

	// Test symmetric property of XOR
	distance1 := xorDistance(id1, id2)
	distance2 := xorDistance(id2, id1)

	if distance1.Cmp(distance2) != 0 {
		t.Errorf("XOR distance is not symmetric")
	}

	// Test identity property of XOR
	identityDistance := xorDistance(id1, id1)
	if identityDistance.Sign() != 0 {
		t.Errorf("XOR distance between identical IDs should be zero")
	}
}

func TestGetKClosest(t *testing.T) {
	meID := []byte("0000000000000000000000")
	me := Triple{
		ID: meID,
		Addr: Address{
			IP:   "127.0.0.1",
			Port: 8000,
		},
	}

	rt := NewRoutingTable(me)

	// Add some contacts with known distances
	contacts := []Triple{
		createTestTriple("1000000000000000000000", 8001), // Far
		createTestTriple("0100000000000000000000", 8002), // Medium
		createTestTriple("0010000000000000000000", 8003), // Close
		createTestTriple("0001000000000000000000", 8004), // Closer
	}

	for _, contact := range contacts {
		rt.addContact(contact)
	}

	// Test getKClosest
	closest := rt.getKClosest("0000000000000000000001")

	// Verify that we get the expected number of contacts
	if len(closest) > BucketSize {
		t.Errorf("getKClosest returned more than BucketSize contacts")
	}

	// If our implementation is complete, we should get contacts sorted by distance
	if len(closest) >= 2 {
		dist1 := xorDistance(closest[0].ID, []byte("0000000000000000000001"))
		dist2 := xorDistance(closest[1].ID, []byte("0000000000000000000001"))

		if dist1.Cmp(dist2) > 0 {
			t.Errorf("Contacts are not sorted by distance")
		}
	}
}

// New test for the getBucketIndexFromDistance function
func TestGetBucketIndexFromDistance(t *testing.T) {
	// Test with distance of 0
	distance := big.NewInt(0)
	index := getBucketIndexFromDistance(distance)
	if index != 0 {
		t.Errorf("Expected bucket index 0 for distance 0, got %d", index)
	}

	// Test with distance of 1 (only rightmost bit set)
	distance = big.NewInt(1)
	index = getBucketIndexFromDistance(distance)
	if index != 0 {
		t.Errorf("Expected bucket index 0 for distance 1, got %d", index)
	}

	// Test with distance of 2 (second bit from right set)
	distance = big.NewInt(2)
	index = getBucketIndexFromDistance(distance)
	if index != 1 {
		t.Errorf("Expected bucket index 1 for distance 2, got %d", index)
	}

	// Test with larger distance
	distance = big.NewInt(128) // 2^7, so 7th bit is set
	index = getBucketIndexFromDistance(distance)
	if index != 7 {
		t.Errorf("Expected bucket index 7 for distance 128, got %d", index)
	}

	// Test with very large distance
	distance = new(big.Int).Exp(big.NewInt(2), big.NewInt(150), nil) // 2^150
	index = getBucketIndexFromDistance(distance)
	if index != 150 {
		t.Errorf("Expected bucket index 150 for distance 2^150, got %d", index)
	}
}

// New test for the insertContact function
func TestInsertContact(t *testing.T) {
	meID := []byte("0000000000000000000000")
	me := Triple{
		ID: meID,
		Addr: Address{
			IP:   "127.0.0.1",
			Port: 8000,
		},
	}

	rt := NewRoutingTable(me)

	// Test inserting a contact with a known distance
	contact := createTestTriple("1000000000000000000000", 8001)
	distance := xorDistance(meID, contact.ID)
	bucketIndex := getBucketIndexFromDistance(distance)

	rt.insertContact(distance, contact)

	// Verify the contact was added to the correct bucket
	found := false
	for _, c := range rt.buckets[bucketIndex].list {
		if bytes.Equal(c.ID, contact.ID) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Contact was not inserted into the correct bucket")
	}

	// Test inserting a contact with invalid bucket index (edge case)
	invalidDist := new(big.Int).Exp(big.NewInt(2), big.NewInt(200), nil) // Should be out of range
	invalidContact := createTestTriple("9999999999999999999999", 9999)

	// This should handle the out-of-range index without panicking
	rt.insertContact(invalidDist, invalidContact)

	// Test inserting a duplicate contact
	rt.insertContact(distance, contact)

	// Check that the bucket still has the contact but didn't add it twice
	count := 0
	for _, c := range rt.buckets[bucketIndex].list {
		if bytes.Equal(c.ID, contact.ID) {
			count++
		}
	}

	if count > 1 {
		t.Errorf("Expected contact to be added only once, found %d times", count)
	}
}

// New test for findNodes method
func TestFindNodes(t *testing.T) {
	meID := []byte("0000000000000000000000")
	me := Triple{
		ID: meID,
		Addr: Address{
			IP:   "127.0.0.1",
			Port: 8000,
		},
	}

	rt := NewRoutingTable(me)

	// Add several contacts with different distances
	contacts := []Triple{
		createTestTriple("1000000000000000000000", 8001), // Far
		createTestTriple("0100000000000000000000", 8002), // Medium
		createTestTriple("0010000000000000000000", 8003), // Close
		createTestTriple("0001000000000000000000", 8004), // Closer
	}

	for _, contact := range contacts {
		rt.addContact(contact)
	}

	// Test finding all nodes
	allNodes := rt.findNodes(big.NewInt(0), 10)
	if len(allNodes) != len(contacts) {
		t.Errorf("Expected to find %d contacts, found %d", len(contacts), len(allNodes))
	}

	// Test limiting the number of nodes returned
	limitedNodes := rt.findNodes(big.NewInt(0), 2)
	if len(limitedNodes) != 2 {
		t.Errorf("Expected to find 2 contacts, found %d", len(limitedNodes))
	}

	// Test finding nodes with a specific distance
	targetID := []byte("0010000000000000000000")
	targetDist := xorDistance(meID, targetID)
	closestNodes := rt.findNodes(targetDist, 1)

	if len(closestNodes) != 1 {
		t.Errorf("Expected to find 1 contact, found %d", len(closestNodes))
	}

	// The closest should be the exact match if we added it
	if len(closestNodes) > 0 && !bytes.Equal(closestNodes[0].ID, targetID) {
		t.Errorf("Expected to find contact with matching ID")
	}
}
