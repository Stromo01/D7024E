package node

import (
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
	fmt.Printf("Found nodes for key %s: %s\n", key, nodes)
	return nodes
}
func (n *Node) IterativeStore(key string, value []byte) {
	var nodes []Triple = n.iterativeFindNode(key)
	fmt.Printf("Storing key %s at nodes: ", key)
	for _, node := range nodes {
		fmt.Printf("%s ", node.Addr.String())
	}
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
		fmt.Printf("New iteration of nodeLookup.Shortlist: \n")
		for _, node := range shortlist {
			fmt.Printf(node.String() + "\n")
		}
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
	responseChan := make(chan queryResult, 1)
	var msgType, responseType string // Determine message type and response handler
	if findValue {
		msgType = "find_value"
		responseType = "find_value_response"
	} else {
		msgType = "find_node"
		responseType = "find_node_response"
	}

	originalHandler := n.handlers[responseType]      // Store original handler
	n.Handle(responseType, func(msg Message) error { // Set temporary handler
		payload := string(msg.Payload)
		if findValue && strings.HasPrefix(payload, "VALUE:") { // Check if it's a value response (only for find_value)
			value := []byte(strings.TrimPrefix(payload, "VALUE:"))
			responseChan <- queryResult{found: true, value: value}
		} else {
			nodes, err := tripleDeserialize(payload) // Parse as nodes
			if err != nil {
				responseChan <- queryResult{found: false, nodes: []Triple{}}
			} else {
				responseChan <- queryResult{found: false, nodes: nodes}
			}
		}
		return nil
	})

	err := n.Send(contact.Addr, msgType, []byte(key)) // Send the query
	if err != nil {
		n.Handle(responseType, originalHandler)
		return queryResult{found: false}
	}

	select { // Wait for response with timeout
	case result := <-responseChan:
		n.Handle(responseType, originalHandler)
		return result
	case <-time.After(5 * time.Second):
		n.Handle(responseType, originalHandler)
		return queryResult{found: false}
	}
}
