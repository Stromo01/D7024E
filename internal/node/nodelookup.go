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
    fmt.Printf("Found nodes for key %s: ", key)
    for _, node := range nodes {
        fmt.Printf("%s ", node.Addr.String())
    }
    fmt.Println()
    return nodes
}

func (n *Node) IterativeStore(key string, value []byte) {
    nodes := n.iterativeFindNode(key)
    fmt.Printf("Storing key %s at nodes: ", key)
    for _, node := range nodes {
        fmt.Printf("%s ", node.Addr.String())
    }
    fmt.Println()
    
    // Store locally first
    n.StoreObject(key, value)
    
    // Store at remote nodes
    for _, node := range nodes {
        payload := fmt.Sprintf("%s:%s", key, string(value))
        err := n.Send(node.Addr, "store", []byte(payload))
        if err != nil {
            fmt.Printf("Failed to store at %s: %v\n", node.Addr.String(), err)
        }
    }
}

// NodeLookup implementation with proper concurrency
func (n *Node) nodeLookup(key string, findValue ...bool) ([]Triple, []byte, bool) {
    isValueSearch := len(findValue) > 0 && findValue[0]
    shortlist := n.routing.GetKClosest(key, K)
    queried := make(map[string]bool)

    for {
        // Select up to Alpha unqueried nodes
        toQuery := make([]Triple, 0, Alpha)
        for _, contact := range shortlist {
            if !queried[contact.Addr.String()] && len(toQuery) < Alpha {
                toQuery = append(toQuery, contact)
                queried[contact.Addr.String()] = true
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
                result := n.queryNode(c, key, isValueSearch)
                resultsChan <- result
            }(contact)
        }

        wg.Wait()
        close(resultsChan)

        // Collect results
        var newNodes []Triple
        for result := range resultsChan {
            if result.found && isValueSearch {
                // Value found! Return immediately
                return shortlist, result.value, true
            }
            newNodes = append(newNodes, result.nodes...)
        }

        // Add new nodes to shortlist
        for _, node := range newNodes {
            if !queried[node.Addr.String()] {
                shortlist = append(shortlist, node)
            }
        }

        // Sort and trim shortlist
        shortlist = n.sortByDistance(key, shortlist)
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

// Fixed queryNode with proper concurrent handler management
func (n *Node) queryNode(contact Triple, key string, findValue bool) queryResult {
    // Create unique message ID for correlation
    msgID := [20]byte{}
    copy(msgID[:], []byte(fmt.Sprintf("%d", time.Now().UnixNano()))[:20])
    
    responseChan := make(chan queryResult, 1)
    
    // Register pending message with unique ID
    n.pendingMu.Lock()
    n.pending[msgID] = make(chan Message, 1)
    responseMsgChan := n.pending[msgID]
    n.pendingMu.Unlock()
    
    // Cleanup pending message registration
    defer func() {
        n.pendingMu.Lock()
        delete(n.pending, msgID)
        n.pendingMu.Unlock()
    }()

    // Determine message type
    var msgType string
    if findValue {
        msgType = "find_value"
    } else {
        msgType = "find_node"
    }

    // Send the query with message ID
    msg := Message{
        ID:          msgID,
        From:        n.Address(),
        FromContact: Triple{ID: n.Id[:], Addr: n.Address(), Port: n.Address().Port},
        To:          contact.Addr,
        Type:        msgType,
        Payload:     []byte(key),
        Network:     n.network,
    }

    err := n.connection.Send(msg)
    if err != nil {
        return queryResult{found: false}
    }

    // Wait for response with timeout
    go func() {
        select {
        case responseMsg := <-responseMsgChan:
            result := n.processQueryResponse(responseMsg, findValue)
            responseChan <- result
        case <-time.After(5 * time.Second):
            responseChan <- queryResult{found: false}
        }
    }()

    select {
    case result := <-responseChan:
        return result
    case <-time.After(6 * time.Second): // Slightly longer than inner timeout
        return queryResult{found: false}
    }
}

// Helper function to process query responses
func (n *Node) processQueryResponse(msg Message, findValue bool) queryResult {
    payload := string(msg.Payload)
    
    if findValue && strings.HasPrefix(payload, "VALUE:") {
        // Value found
        value := []byte(strings.TrimPrefix(payload, "VALUE:"))
        return queryResult{found: true, value: value}
    }
    
    // Parse as nodes
    nodes, err := tripleDeserialize(payload)
    if err != nil {
        return queryResult{found: false, nodes: []Triple{}}
    }
    
    return queryResult{found: false, nodes: nodes}
}
