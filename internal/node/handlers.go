package node

import (
	"fmt"
	"strings"

	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

func (node *Node) handleStore(msg Message) error {
	parts := strings.SplitN(string(msg.Payload), ":", 2)
	if len(parts) == 2 {
		key := parts[0]
		value := []byte(parts[1])

		node.StoreObject(key, value)
		fmt.Printf("Node %s stored object with key %s from %s\n",
			node.Address().String(), key, msg.From.String())

		// Add the sender to routing table
		if len(msg.FromContact.ID) > 0 {
			node.routing.AddContact(msg.FromContact)
		}
	}
	return nil
}

func (node *Node) handlePing(msg Message) error {
	fmt.Printf("Node %s received PING from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)

	// Use the address from FromContact, not msg.From
	contactToAdd := Triple{
		ID:   msg.FromContact.ID,
		Addr: msg.FromContact.Addr, // Use this instead of msg.From
		Port: msg.FromContact.Port,
	}
	node.routing.AddContact(contactToAdd)
	return node.Send(msg.FromContact.Addr, MsgPong, []byte("pong"))
}

func (node *Node) handlePong(msg Message) error {
	fmt.Printf("Node %s received pong from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)

	contactToAdd := Triple{
		ID:   msg.FromContact.ID,
		Addr: msg.FromContact.Addr,
		Port: msg.FromContact.Port,
	}

	node.routing.AddContact(contactToAdd)
	return nil
}

func (node *Node) handleFindNode(msg Message) error {
	fmt.Printf("Node %s received find_node from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)

	// Expect payload as "key"
	key := string(msg.Payload)
	closest := node.routing.GetKClosest(key, K)
	var respPayload = tripleSerialize(closest)
	return node.Send(msg.From, "find_node_response", []byte(respPayload))
}

func (node *Node) handleFindNodeResponse(msg Message) error {
	// Expect payload as "addr1:port1:id1,addr2:port2:id2,..."
	fmt.Printf("Node %s received find_node_response from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)

	payload := string(msg.Payload)
	if payload != "" {
		triples, err := tripleDeserialize(payload)
		if err != nil {
			return fmt.Errorf("invalid find_node_response payload: %v", err)
		}
		for _, t := range triples {
			node.routing.AddContact(t)
		}
	}
	return nil
}

func (node *Node) handleFindValue(msg Message) error {
	fmt.Printf("Node %s received find_value from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)
	key := string(msg.Payload)

	// Add the sender to routing table
	if len(msg.FromContact.ID) > 0 {
		node.routing.AddContact(msg.FromContact)
	}

	// Check if we have the value
	if val, ok := node.FindObjectLocally(key); ok {
		return node.Send(msg.From, "find_value_response", []byte("VALUE:"+string(val)))
	} else {
		// Return closest nodes
		closest := node.routing.GetKClosest(key, K)
		respPayload := tripleSerialize(closest)
		return node.Send(msg.From, "find_value_response", []byte(respPayload))
	}
}
