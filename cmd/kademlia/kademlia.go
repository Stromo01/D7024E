package kademlia

// Kademlia represents a Kademlia node (adapted from reference repo)
type Kademlia struct {
	RoutingTable *RoutingTable
	Network      Network
}

func NewKademlia(rt *RoutingTable, net Network) *Kademlia {
	return &Kademlia{
		RoutingTable: rt,
		Network:      net,
	}
}

// LookupContact performs the iterative lookup process to find the k closest contacts to the target.
func (k *Kademlia) LookupContact(target *KademliaID) []Contact {
	lookup := NewLookup(k.RoutingTable, k.Network, target)
	return lookup.Start()
}
