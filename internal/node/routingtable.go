package node

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"sync"

	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

const IDLength = 20

type RoutingTable struct {
	Me      Triple
	Buckets [IDLength * 8]*Bucket
	mu      sync.RWMutex
}

func NewRoutingTable(Me Triple) *RoutingTable {
	rt := &RoutingTable{
		Me: Me,
	}
	for i := range rt.Buckets {
		rt.Buckets[i] = &Bucket{
			List: make([]*Triple, 0, K),
		}
	}
	return rt
}

func (rt *RoutingTable) GetKClosest(key string, K int) []Triple {
	// Read lock for accessing buckets
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	keyBytes := []byte(key)
	type distTriple struct {
		dist    *big.Int
		contact Triple
	}
	var all []distTriple

	for _, bucket := range rt.Buckets {
		for _, contact := range bucket.List {
			d := XorDistance(keyBytes, contact.ID)
			all = append(all, distTriple{dist: d, contact: *contact})
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].dist.Cmp(all[j].dist) < 0
	})

	var result []Triple
	for i := 0; i < len(all) && i < K; i++ {
		result = append(result, all[i].contact)
	}
	return result
}

func (rt *RoutingTable) AddContact(contact Triple) {
	rt.mu.RLock() // Read lock for getting bucket

	if bytes.Equal(contact.ID, rt.Me.ID) {
		return // Don't add ourselves
	}
	for _, bucket := range rt.Buckets { //Dont add if it already exists
		for _, c := range bucket.List {
			if bytes.Equal(c.ID, contact.ID) {
				return
			}
		}
	}
	fmt.Printf("Adding contact %s (ID: %x) to routing table\n", contact.Addr.String(), contact.ID)
	bucketIndex := rt.GetBucketIndex(contact.ID)
	bucket := rt.Buckets[bucketIndex]
	rt.mu.RUnlock()

	// Bucket handles its own locking
	bucket.AddContact(contact)
}

func (rt *RoutingTable) RemoveContact(contact Triple) {
	bucketIndex := rt.GetBucketIndex(contact.ID)
	bucket := rt.Buckets[bucketIndex]
	bucket.RemoveContact(contact)
	// bucket := rt.Buckets[bucketI] // Uncomment and use as needed
	// rt.insertContact(distance, contact) // Update as needed

}

func (routingTable *RoutingTable) GetBucketIndex(nodeID []byte) int {
	distance := XorDistance(routingTable.Me.ID, nodeID)
	// If distance is 0 (identical IDs), return bucket 0
	if distance.Sign() == 0 {
		return 0
	}

	// Calculate bucket index based on the position of the most significant bit
	// BitLen() returns the number of bits needed to represent the number
	// For Kademlia, we want bucket 0 for the closest nodes (smallest distances)
	bitLen := distance.BitLen()
	index := bitLen - 1

	// Ensure index is within valid range [0, IDLength*8-1]
	if index < 0 {
		index = 0
	}
	if index >= IDLength*8 {
		index = IDLength*8 - 1
	}
	return index
}
