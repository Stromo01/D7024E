package main

import "math/big"

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

func (routing *RoutingTable) getKClosest(key string) []Triple {
	var distance = xorDistance(routing.me.ID, []byte(key))
	return routing.findNodes(distance, BucketSize)
}

func (rt *RoutingTable) findNodes(distance *big.Int, count int) []Triple { //TODO: implement
	var result []Triple
	return result
}

func (rt *RoutingTable) addContact(contact Triple) {
	var distance = xorDistance(rt.me.ID, contact.ID)
	rt.insertContact(distance, contact)
}

func (rt *RoutingTable) insertContact(distance *big.Int, contact Triple) { //TODO: implement

}
