package kademlia

import (
	"fmt"
	"sort"
)

// Address represents a node's network address (shared type for net.go)
type Address struct {
	IP   string
	Port int // 1-65535
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

// Contact definition (adapted from reference repo)
type Contact struct {
	ID       *KademliaID
	Address  string
	distance *KademliaID
}

func NewContact(id *KademliaID, address string) Contact {
	return Contact{id, address, nil}
}

func (contact *Contact) CalcDistance(target *KademliaID) {
	contact.distance = contact.ID.CalcDistance(target)
}

func (contact *Contact) Less(otherContact *Contact) bool {
	return contact.distance.Less(otherContact.distance)
}

func (contact *Contact) String() string {
	return fmt.Sprintf(`contact("%s", "%s")`, contact.ID, contact.Address)
}

// ContactCandidates stores an array of Contacts
type ContactCandidates struct {
	contacts []Contact
}

func (candidates *ContactCandidates) Append(contacts []Contact) {
	candidates.contacts = append(candidates.contacts, contacts...)
}

func (candidates *ContactCandidates) GetContacts(count int) []Contact {
	if count > len(candidates.contacts) {
		count = len(candidates.contacts)
	}
	return candidates.contacts[:count]
}

func (candidates *ContactCandidates) Sort() {
	sort.Sort(candidates)
}

func (candidates *ContactCandidates) Len() int { return len(candidates.contacts) }
func (candidates *ContactCandidates) Swap(i, j int) {
	candidates.contacts[i], candidates.contacts[j] = candidates.contacts[j], candidates.contacts[i]
}
func (candidates *ContactCandidates) Less(i, j int) bool {
	return candidates.contacts[i].Less(&candidates.contacts[j])
}
