// lookup.go: Iterative lookup logic placeholder

package kademlia

import "sync"

const alpha = 3

// Lookup holds the state for a single iterative lookup process.
type Lookup struct {
	shortlist    *ContactCandidates
	queried      map[string]bool // stringified KademliaID
	routingTable *RoutingTable
	network      Network
	target       *KademliaID
}

func NewLookup(rt *RoutingTable, net Network, target *KademliaID) *Lookup {
	return &Lookup{
		shortlist:    &ContactCandidates{},
		queried:      make(map[string]bool),
		routingTable: rt,
		network:      net,
		target:       target,
	}
}

func (l *Lookup) Start() []Contact {
	initialContacts := l.routingTable.FindClosestContacts(l.target, alpha)
	l.shortlist.Append(initialContacts)

	var closestContact *Contact
	if l.shortlist.Len() > 0 {
		closestContact = &l.shortlist.GetContacts(1)[0]
	}

	for {
		contactsToQuery := l.getUnqueriedContacts(alpha)
		if len(contactsToQuery) == 0 {
			break
		}
		newContacts := l.queryContacts(contactsToQuery)
		l.shortlist.Append(newContacts)
		l.shortlist.Sort()
		if l.shortlist.Len() > 0 && (closestContact == nil || l.shortlist.GetContacts(1)[0].Less(closestContact)) {
			closestContact = &l.shortlist.GetContacts(1)[0]
		} else {
			remainingToQuery := l.getUnqueriedContacts(BucketSize)
			if len(remainingToQuery) > 0 {
				newContacts := l.queryContacts(remainingToQuery)
				l.shortlist.Append(newContacts)
				l.shortlist.Sort()
			}
			break
		}
	}
	return l.shortlist.GetContacts(BucketSize)
}

func (l *Lookup) getUnqueriedContacts(count int) []Contact {
	var contacts []Contact
	for _, contact := range l.shortlist.GetContacts(l.shortlist.Len()) {
		if len(contacts) >= count {
			break
		}
		if !l.queried[contact.ID.String()] {
			contacts = append(contacts, contact)
		}
	}
	return contacts
}

func (l *Lookup) queryContacts(contacts []Contact) []Contact {
	var newContacts []Contact
	var wg sync.WaitGroup
	resultsChan := make(chan []Contact, len(contacts))
	for _, contact := range contacts {
		l.queried[contact.ID.String()] = true
		wg.Add(1)
		go func(c Contact) {
			defer wg.Done()
			c.CalcDistance(l.target)
			// TODO: implement network FindNode RPC
			// foundContacts, err := l.network.FindNode(&c, l.target)
			// if err == nil {
			//     resultsChan <- foundContacts
			// }
			// For now, just return empty
			resultsChan <- nil
		}(contact)
	}
	wg.Wait()
	close(resultsChan)
	for res := range resultsChan {
		if res != nil {
			newContacts = append(newContacts, res...)
		}
	}
	return newContacts
}
