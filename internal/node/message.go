package node

import (
	"log"

	. "github.com/eislab-cps/go-template/internal/network"
)

func (n *Node) HandleMessageConcurrently(msg Message) {

	// Try to acquire semaphore (non-blocking)

	select {
	case n.semaphore <- struct{}{}: // Acquired semaphore
		// Add to WaitGroup before starting goroutine
		n.messageWG.Add(1)
		// Handle message in separate goroutine
		go func(message Message) {
			defer func() {
				// Release semaphore and mark as done
				<-n.semaphore
				n.messageWG.Done()
			}()
			// Handle the message
			n.handleMessage(message)
		}(msg)

	default:
		// All handlers busy, handle synchronously to prevent blocking
		log.Printf("All %d handlers busy, handling message synchronously", n.maxConcurrent)
		n.handleMessage(msg)
	}

}

func (n *Node) handleMessage(msg Message) {
	// Check if this is a pending correlation response
	n.pendingMu.Lock()
	if ch, ok := n.pending[msg.ID]; ok {
		select {
		case ch <- msg:

		default:

		}
		delete(n.pending, msg.ID)
		n.pendingMu.Unlock()
		return
	}

	n.pendingMu.Unlock()
	msgType := msg.Type
	if msgType == "" {
		msgType = "default"
	}

	// Get handler (thread-safe read)

	n.mu.RLock()
	handler, exists := n.handlers[msgType]
	if !exists {
		handler, exists = n.handlers["default"]
	}
	n.mu.RUnlock()

	// Execute handler
	if exists && handler != nil {
		if err := handler(msg); err != nil {
			log.Printf("Handler error from %s: %v", msg.From.String(), err)
		}

	} else {
		log.Printf("No handler for msg type %q from %s", msgType, msg.From.String())

	}

}
