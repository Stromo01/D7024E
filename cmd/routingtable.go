package main

import (
	"math/big"
	"sort"
)

const BucketSize = 20
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
	bucketIndex := rt.getBucketIndex(contact.ID)
	bucket := rt.buckets[bucketIndex]
	bucket.AddContact(contact)
	// bucket := rt.buckets[bucketI] // Uncomment and use as needed
	// rt.insertContact(distance, contact) // Update as needed

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
	index := distance.BitLen() - 1
	if index < 0 {
		index = 0
	}
	return index
}

func (rt *RoutingTable) insertContact(distance *big.Int, contact Triple) { //TODO: implement

}
