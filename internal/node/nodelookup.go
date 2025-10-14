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

func dedupByID(in []Triple) []Triple {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]Triple, 0, len(in))
	for _, t := range in {
		k := fmt.Sprintf("%x", t.ID)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, t)
	}
	return out
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

// NodeLookup
func (n *Node) nodeLookup(key string, findValue ...bool) ([]Triple, []byte, bool) {
	isValueSearch := len(findValue) > 0 && findValue[0]

	shortlist := n.routing.GetKClosest(key, K) // List of known closest nodes
	shortlist = dedupByID(shortlist)           // Remove duplicates

	queried := make(map[string]bool, len(shortlist)) // Track queried nodes by address string

	for {
		fmt.Printf("----------\n")
		fmt.Printf("New iteration of nodeLookup.Shortlist:\n")
		for _, node := range shortlist {
			fmt.Printf("%s\n", node.String())
		}
		fmt.Printf("----------\n")

		toQuery := make([]Triple, 0, Alpha)
		for _, c := range shortlist { // Select up to Alpha unqueried nodes
			if !queried[c.Addr.String()] && len(toQuery) < Alpha {
				toQuery = append(toQuery, c)
				queried[c.Addr.String()] = true
			}
		}

		if len(toQuery) == 0 {
			break
		}

		// Query in parallel
		var wg sync.WaitGroup
		resultsChan := make(chan queryResult, len(toQuery))

		for _, contact := range toQuery {
			wg.Add(1)
			go func(c Triple) {
				defer wg.Done()
				resultsChan <- n.queryNode(c, key, isValueSearch)
			}(contact)
		}

		wg.Wait()
		close(resultsChan)

		// Collect newly learned nodes and value (if any)
		var (
			newNodes []Triple
			value    []byte
			found    bool
		)

		for result := range resultsChan {
			if result.found { // Value found
				return shortlist, result.value, true
			}
			if len(result.nodes) > 0 { // New nodes learned
				newNodes = append(newNodes, result.nodes...)
			}
			value = result.value
			found = result.found
		}
		if found { //Needed for var definition
			return shortlist, value, true
		}

		// Merge nodes learned from responses
		if len(newNodes) > 0 {
			shortlist = append(shortlist, newNodes...)
		}

		// Also merge whatever the handlers might have added to the routing table
		// during this round to keep discovery progressing even if a specific
		// correlation-by-ID missed.
		rtClosest := n.routing.GetKClosest(key, K*2)
		if len(rtClosest) > 0 {
			shortlist = append(shortlist, rtClosest...)
		}

		// De-duplicate by node ID
		shortlist = dedupByID(shortlist)

		// Re-sort by distance, then trim to K
		shortlist = n.sortByDistance(key, shortlist)
		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}

	return shortlist, nil, false
}

type queryResult struct {
	found bool
	value []byte
	nodes []Triple
}

func (n *Node) queryNode(contact Triple, key string, findValue bool) queryResult {
	msgType := "find_node"
	if findValue {
		msgType = "find_value"
	}

	var correlationID [20]byte // Correlation ID
	_, _ = rand.Read(correlationID[:])

	// Register waiter
	respCh := make(chan Message, 1)
	n.pendingMu.Lock()
	if n.pending == nil {
		n.pending = make(map[[20]byte]chan Message)
	}
	n.pending[correlationID] = respCh
	n.pendingMu.Unlock()

	defer func() {
		n.pendingMu.Lock()
		delete(n.pending, correlationID) // Clean up when done
		n.pendingMu.Unlock()
	}()

	// Send request with ID + Type over the wire
	out := Message{
		ID:          correlationID,
		Type:        msgType,
		From:        n.Addr,
		FromContact: Triple{ID: n.Id[:], Addr: n.Addr, Port: n.Addr.Port},
		To:          contact.Addr,
		Payload:     []byte(key),
		Network:     n.network,
	}
	if err := n.connection.Send(out); err != nil {
		return queryResult{found: false}
	}

	select { // Wait for the correlated response
	case resp := <-respCh:
		return n.processQueryResponse(resp, findValue)
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
