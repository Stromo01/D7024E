package node

import (
	"fmt"
	"strings"

	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

func (node *Node) handleStore(msg Message) error {
	parts := strings.SplitN(string(msg.Payload), ":", 2)
	fmt.Printf("Storing %s", parts)
	node.routing.AddContact(msg.FromContact)
	if len(parts) == 2 {
		key := parts[0]
		value := []byte(parts[1])

		node.StoreObject(key, value)
		fmt.Printf("Node %s stored object with key %s from %s\n",
			node.Address().String(), key, msg.From.String())
	}
	return nil
}

func (node *Node) handlePing(msg Message) error {
	fmt.Printf("Node %s received PING from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)

	node.routing.AddContact(msg.FromContact)
	return node.Send(msg.FromContact.Addr, MsgPong, []byte("pong"))
}

func (node *Node) handlePong(msg Message) error {
	fmt.Printf("Node %s received pong from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)
	node.routing.AddContact(msg.FromContact)
	return nil
}

func (node *Node) handleFindNode(msg Message) error {
	fmt.Printf("Node %s received find_node from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)
	node.routing.AddContact(msg.FromContact)
	key := string(msg.Payload)
	closest := node.routing.GetKClosest(key, K)
	var respPayload = TripleSerialize(closest)

	// Send response with same correlation ID
	responseMsg := Message{
		ID:          msg.ID, // Keep same correlation ID
		From:        node.Addr,
		FromContact: Triple{ID: node.Id[:], Addr: node.Addr, Port: node.Addr.Port},
		To:          msg.From,
		Type:        "find_node_response",
		Payload:     []byte(respPayload),
		Network:     node.network,
	}

	return node.connection.Send(responseMsg)
}

func (node *Node) handleFindNodeResponse(msg Message) error {
	fmt.Printf("Node %s received find_node_response from %s (ID: %x)\n",
		node.Address().String(),
		msg.From.String(),
		msg.FromContact.ID)

	node.routing.AddContact(msg.FromContact)

	// Check if this is a correlated response
	node.pendingMu.Lock()
	if responseChan, exists := node.pending[msg.ID]; exists {
		select {
		case responseChan <- msg:
			fmt.Printf("Routed find_node response to waiting query\n")
		default:
			fmt.Printf("Response channel full, dropping message\n")
		}
		node.pendingMu.Unlock()
		return nil
	}
	node.pendingMu.Unlock()

	// Handle uncorrelated response
	payload := string(msg.Payload)
	if payload != "" {
		triples, err := TripleDeserialize(payload)
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
	node.routing.AddContact(msg.FromContact)

	var responsePayload []byte
	if val, ok := node.FindObjectLocally(key); ok {
		fmt.Printf("Node %s found value for key %s locally\n", node.Address().String(), key)
		responsePayload = []byte("VALUE:" + string(val))
	} else {
		closest := node.routing.GetKClosest(key, K)
		respPayload := TripleSerialize(closest)
		fmt.Printf("Value not found, Node %s found closest nodes for key %s: %s\n", node.Address().String(), key, respPayload)
		responsePayload = []byte(respPayload)
	}

	// Send response with same correlation ID
	responseMsg := Message{
		ID:          msg.ID, // Keep same correlation ID
		From:        node.Addr,
		FromContact: Triple{ID: node.Id[:], Addr: node.Addr, Port: node.Addr.Port},
		To:          msg.From,
		Type:        "find_value_response",
		Payload:     responsePayload,
		Network:     node.network,
	}

	return node.connection.Send(responseMsg)
}

func (node *Node) handleFindValueResponse(msg Message) error {
	fmt.Printf("Node %s received find_value_response: %s\n",
		node.Address().String(), string(msg.Payload))

	// Add sender to routing table
	correctedContact := Triple{
		ID:   msg.FromContact.ID,
		Addr: msg.From,
		Port: msg.From.Port,
	}
	node.routing.AddContact(correctedContact)

	// Check if this is a correlated response
	node.pendingMu.Lock()
	if responseChan, exists := node.pending[msg.ID]; exists {
		select {
		case responseChan <- msg:
			fmt.Printf("Routed response to waiting query\n")
		default:
			fmt.Printf("Response channel full, dropping message\n")
		}
		node.pendingMu.Unlock()
		return nil
	}
	node.pendingMu.Unlock()

	// Handle uncorrelated response (from original handlers)
	payload := string(msg.Payload)
	if strings.HasPrefix(payload, "VALUE:") {
		fmt.Printf("Received uncorrelated VALUE response: %s\n", payload)
		return nil
	}

	// Parse and add nodes to routing table
	fmt.Printf("Received uncorrelated node list, parsing contacts...\n")
	triples, err := TripleDeserialize(payload)
	if err != nil {
		return fmt.Errorf("invalid find_value_response payload: %v", err)
	}

	fmt.Printf("Parsed %d contacts from uncorrelated response\n", len(triples))
	for i, triple := range triples {
		fmt.Printf("  [%d] Adding contact: ID=%x, Addr=%s\n",
			i, triple.ID, triple.Addr.String())
		node.routing.AddContact(triple)
	}

	return nil
}
