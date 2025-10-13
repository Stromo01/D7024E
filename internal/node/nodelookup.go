package node

import (
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	. "github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

// Iterative Functions:
func (n *Node) iterativeFindValue(key string) ([]byte, bool) {
	_, value, found := n.nodeLookup(key, true)
	return value, found
}

func (n *Node) iterativeFindNode(key string) []Triple {
	nodes, _, _ := n.nodeLookup(key, false)
	fmt.Printf("----------\n")
	fmt.Printf("Found nodes for key %s: \n", key)
	for _, node := range nodes {
		fmt.Printf(" - %s\n", node.Addr.String())
	}
	fmt.Printf("----------\n")
	return nodes
}
func (n *Node) IterativeStore(key string, value []byte) {
	var nodes []Triple = n.iterativeFindNode(key)
	fmt.Printf("----------\n")
	fmt.Printf("Storing key %s at nodes: ", key)
	for _, node := range nodes {
		fmt.Printf("%s ", node.Addr.String())
	}
	fmt.Printf("----------\n")
	for _, node := range nodes {
		payload := fmt.Sprintf("%s:%s", key, string(value))
		err := n.Send(node.Addr, "store", []byte(payload))
		if err != nil {
			fmt.Printf("Failed to store at %s: %v\n", node.Addr.String(), err)
		} else {
			fmt.Printf("Successfully sent store request to %s\n", node.Addr.String())
		}
	}
}

// NodeLookup
func (n *Node) nodeLookup(key string, findValue ...bool) ([]Triple, []byte, bool) {
	isValueSearch := len(findValue) > 0 && findValue[0]
	shortlist := n.routing.GetKClosest(key, K)
	queried := make(map[string]bool)

	for {
		fmt.Printf("----------\n")
		fmt.Printf("New iteration of nodeLookup.Shortlist: \n")
		for _, node := range shortlist {
			fmt.Printf(node.String() + "\n")
		}
		fmt.Printf("----------\n")
		toQuery := make([]Triple, 0, Alpha) // Select up to Alpha unqueried nodes
		for _, contact := range shortlist {
			if !queried[contact.Addr.String()] && len(toQuery) < Alpha {
				toQuery = append(toQuery, contact)
				queried[contact.Addr.String()] = true
			}
		}

		if len(toQuery) == 0 {
			break
		}

		var wg sync.WaitGroup // Query in parallel
		resultsChan := make(chan queryResult, len(toQuery))

		for _, contact := range toQuery {
			wg.Add(1)
			go func(c Triple) {
				defer wg.Done()
				result := n.queryNode(c, key, isValueSearch)
				resultsChan <- result
			}(contact)
		}

		wg.Wait()
		close(resultsChan)

		var newNodes []Triple // Collect new nodes
		for result := range resultsChan {
			if result.found { // Value found! Return immediately
				return shortlist, result.value, true
			}
			newNodes = append(newNodes, result.nodes...)
		}

		for _, node := range newNodes { // Add new nodes to shortlist
			if !queried[node.Addr.String()] {
				shortlist = append(shortlist, node)
			}
		}

		shortlist = n.sortByDistance(key, shortlist) // Sort and trim
		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}

	return shortlist, nil, false
}

func (n *Node) sortByDistance(key string, nodes []Triple) []Triple {
	keyBytes := []byte(key)

	sort.Slice(nodes, func(i, j int) bool {
		distI := XorDistance(keyBytes, nodes[i].ID)
		distJ := XorDistance(keyBytes, nodes[j].ID)
		return distI.Cmp(distJ) < 0
	})

	return nodes
}

type queryResult struct {
	found bool
	value []byte
	nodes []Triple
}

func (n *Node) queryNode(contact Triple, key string, findValue bool) queryResult {
	var msgType string
	if findValue {
		msgType = "find_value"
	} else {
		msgType = "find_node"
	}

	// Generate unique correlation ID for this query
	var correlationID [20]byte
	rand.Read(correlationID[:])

	// Set up response channel before sending
	responseChan := make(chan Message, 1)
	n.pendingMu.Lock()
	n.pending[correlationID] = responseChan
	n.pendingMu.Unlock()

	// Clean up on function exit
	defer func() {
		n.pendingMu.Lock()
		delete(n.pending, correlationID)
		n.pendingMu.Unlock()
	}()

	// Create message with correlation ID
	msg := Message{
		ID:          correlationID,
		From:        n.Addr,
		FromContact: Triple{ID: n.Id[:], Addr: n.Addr, Port: n.Addr.Port},
		To:          contact.Addr,
		Type:        msgType,
		Payload:     []byte(key),
		Network:     n.network,
	}

	// Send the query
	err := n.connection.Send(msg)
	if err != nil {
		return queryResult{found: false}
	}

	// Wait for correlated response
	select {
	case responseMsg := <-responseChan:
		return n.processQueryResponse(responseMsg, findValue)
	case <-time.After(5 * time.Second):
		fmt.Printf("Query to %s timed out\n", contact.Addr.String())
		return queryResult{found: false}
	}
}

func (n *Node) processQueryResponse(msg Message, findValue bool) queryResult {
	payload := string(msg.Payload)
	fmt.Printf("Processing query response: %s\n", payload)

	if findValue && strings.HasPrefix(payload, "VALUE:") {
		value := []byte(strings.TrimPrefix(payload, "VALUE:"))
		return queryResult{found: true, value: value}
	}

	// Parse as node list
	nodes, err := tripleDeserialize(payload)
	if err != nil {
		fmt.Printf("Error deserializing nodes: %v\n", err)
		return queryResult{found: false, nodes: []Triple{}}
	}

	fmt.Printf("Parsed %d nodes from response\n", len(nodes))
	return queryResult{found: false, nodes: nodes}
}
