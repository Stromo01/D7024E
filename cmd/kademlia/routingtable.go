package kademlia

const BucketSize = 20
const IDLength = 20

// RoutingTable definition (adapted from reference repo)
type RoutingTable struct {
	me      Contact
	buckets [IDLength * 8]*Bucket
}

func NewRoutingTable(me Contact) *RoutingTable {
	rt := &RoutingTable{}
	for i := 0; i < IDLength*8; i++ {
		rt.buckets[i] = NewBucket()
	}
	rt.me = me
	return rt
}

func (rt *RoutingTable) AddContact(contact Contact, net Network) {
	bucketIndex := rt.getBucketIndex(contact.ID)
	bucket := rt.buckets[bucketIndex]
	bucket.AddContact(contact, net)
}

func (rt *RoutingTable) FindClosestContacts(target *KademliaID, count int) []Contact {
	var candidates ContactCandidates
	bucketIndex := rt.getBucketIndex(target)
	bucket := rt.buckets[bucketIndex]
	candidates.Append(bucket.GetContactAndCalcDistance(target))
	for i := 1; (bucketIndex-i >= 0 || bucketIndex+i < IDLength*8) && candidates.Len() < count; i++ {
		if bucketIndex-i >= 0 {
			bucket = rt.buckets[bucketIndex-i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
		if bucketIndex+i < IDLength*8 {
			bucket = rt.buckets[bucketIndex+i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
	}
	candidates.Sort()
	if count > candidates.Len() {
		count = candidates.Len()
	}
	return candidates.GetContacts(count)
}

func (rt *RoutingTable) getBucketIndex(id *KademliaID) int {
	distance := id.CalcDistance(rt.me.ID)
	for i := 0; i < IDLength; i++ {
		for j := 0; j < 8; j++ {
			if (distance[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return IDLength*8 - 1
}
