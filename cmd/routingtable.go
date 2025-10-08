package main

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
)

const BucketSize = 8
const IDLength = 20

type RoutingTable struct {
	me      Triple
	buckets [IDLength * 8]*Bucket
}

func NewRoutingTable(me Triple) *RoutingTable {
	rt := &RoutingTable{
		me: me,
	}
	for i := range rt.buckets {
		rt.buckets[i] = &Bucket{
			list: make([]*Triple, 0, BucketSize),
		}
	}
	return rt
}

func (rt *RoutingTable) getKClosest(key string, K int) []Triple {
	keyBytes := []byte(key)
	type distTriple struct {
		dist    *big.Int
		contact Triple
	}
	var all []distTriple

	for _, bucket := range rt.buckets {
		for _, contact := range bucket.list {
			d := xorDistance(keyBytes, contact.ID)
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

func (rt *RoutingTable) addContact(contact Triple) {
	fmt.Printf("Adding contact %s (ID: %x) to routing table\n", contact.Addr.String(), contact.ID)
	if bytes.Equal(contact.ID, rt.me.ID) {
		return // Don't add ourselves
	}

	bucketIndex := rt.getBucketIndex(contact.ID)
	bucket := rt.buckets[bucketIndex]
	bucket.AddContact(contact)
}

func (rt *RoutingTable) RemoveContact(contact Triple) {
	bucketIndex := rt.getBucketIndex(contact.ID)
	bucket := rt.buckets[bucketIndex]
	bucket.RemoveContact(contact)
	// bucket := rt.buckets[bucketI] // Uncomment and use as needed
	// rt.insertContact(distance, contact) // Update as needed

}

func (routingTable *RoutingTable) getBucketIndex(nodeID []byte) int {
	distance := xorDistance(routingTable.me.ID, nodeID)
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

func (rt *RoutingTable) insertContact(distance *big.Int, contact Triple) { //TODO: implement

}
