package network_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/eislab-cps/go-template/internal/network"
	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

func TestNewUDPNetwork(t *testing.T) {
	net := network.NewUDPNetwork()
	if net == nil {
		t.Fatal("NewUDPNetwork returned nil")
	}
}

func TestUDPNetworkListenAndDial(t *testing.T) {
	net := network.NewUDPNetwork()

	// Test Listen
	addr := Address{IP: "127.0.0.1", Port: 0}
	listener, err := net.Listen(addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	actualAddr := listener.LocalAddr()
	if actualAddr == nil {
		t.Fatal("LocalAddr returned nil")
	}

	// Test Dial
	dialAddr := network.AddressFromNetAddr(actualAddr)
	conn, err := net.Dial(dialAddr)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()
}

func TestMessageSendRecv(t *testing.T) {
	net := network.NewUDPNetwork()

	// Setup listener
	addr := Address{IP: "127.0.0.1", Port: 0}
	listener, err := net.Listen(addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	listenerAddr := network.AddressFromNetAddr(listener.LocalAddr())

	// Setup sender
	sender, err := net.Dial(listenerAddr)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer sender.Close()

	senderAddr := network.AddressFromNetAddr(sender.LocalAddr())

	// Test message - ADD Type field
	testPayload := []byte("test message")
	testContact := Triple{
		ID:   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		Addr: senderAddr,
		Port: senderAddr.Port,
	}

	msg := network.Message{
		From:        senderAddr,
		FromContact: testContact,
		To:          listenerAddr,
		Type:        "test", // ADD THIS LINE
		Payload:     testPayload,
		Network:     net,
	}

	// Send message
	go func() {
		if err := sender.Send(msg); err != nil {
			t.Errorf("Failed to send message: %v", err)
		}
	}()

	// Receive message
	received, err := listener.Recv()
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	if !bytes.Equal(received.Payload, testPayload) {
		t.Errorf("Payload mismatch: got %s, want %s", received.Payload, testPayload)
	}

	if received.From.IP != senderAddr.IP || received.From.Port != senderAddr.Port {
		t.Errorf("From address mismatch: got %v, want %v", received.From, senderAddr)
	}

	// ADD: Check Type field
	if received.Type != "test" {
		t.Errorf("Type mismatch: got %s, want test", received.Type)
	}
}

func TestNetworkPartition(t *testing.T) {
	net := network.NewUDPNetwork()

	addr1 := Address{IP: "127.0.0.1", Port: 8001}
	addr2 := Address{IP: "127.0.0.1", Port: 8002}

	// Partition network
	net.Partition([]Address{addr1}, []Address{addr2})

	// Try to dial partitioned address
	_, err := net.Dial(addr1)
	if err == nil {
		t.Error("Expected error when dialing partitioned address")
	}

	// Heal network
	net.Heal()

	// Should work after healing (though may fail for other reasons like no listener)
	_, err = net.Dial(addr1)
	// We don't check for success here since there's no listener, just that partition is gone
}

func TestSendPing(t *testing.T) {
	net := network.NewUDPNetwork()

	// Setup listener
	addr := Address{IP: "127.0.0.1", Port: 0}
	listener, err := net.Listen(addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	listenerAddr := network.AddressFromNetAddr(listener.LocalAddr())
	fromAddr := Address{IP: "127.0.0.1", Port: 9999}

	// Send ping in goroutine
	go func() {
		if err := network.SendPing(net, fromAddr, listenerAddr); err != nil {
			t.Errorf("Failed to send ping: %v", err)
		}
	}()

	// Receive ping
	msg, err := listener.Recv()
	if err != nil {
		t.Fatalf("Failed to receive ping: %v", err)
	}

	// FIX: Check Type field instead of payload format
	if msg.Type != network.MsgPing {
		t.Errorf("Ping type mismatch: got %s, want %s", msg.Type, network.MsgPing)
	}

	if string(msg.Payload) != "ping" {
		t.Errorf("Ping payload mismatch: got %s, want ping", msg.Payload)
	}
}

func TestMessageReply(t *testing.T) {
	net := network.NewUDPNetwork()

	// Setup original sender (will receive reply)
	senderAddr := Address{IP: "127.0.0.1", Port: 0}
	sender, err := net.Listen(senderAddr)
	if err != nil {
		t.Fatalf("Failed to setup sender listener: %v", err)
	}
	defer sender.Close()

	actualSenderAddr := network.AddressFromNetAddr(sender.LocalAddr())

	// Create a message as if received
	receivedMsg := network.Message{
		From:    actualSenderAddr,
		To:      Address{IP: "127.0.0.1", Port: 8888},
		Network: net,
	}

	// Send reply in goroutine
	go func() {
		if err := receivedMsg.ReplyString(network.MsgPong, "pong response"); err != nil {
			t.Errorf("Failed to send reply: %v", err)
		}
	}()

	// Receive reply
	reply, err := sender.Recv()
	if err != nil {
		t.Fatalf("Failed to receive reply: %v", err)
	}

	// FIX: Check Type field and Payload separately (not combined)
	if reply.Type != network.MsgPong {
		t.Errorf("Reply type mismatch: got %s, want %s", reply.Type, network.MsgPong)
	}

	if string(reply.Payload) != "pong response" {
		t.Errorf("Reply payload mismatch: got %s, want pong response", reply.Payload)
	}
}

func TestGetLocalIP(t *testing.T) {
	ip, err := network.GetLocalIP()
	if err != nil {
		t.Fatalf("Failed to get local IP: %v", err)
	}

	if ip == "" {
		t.Error("GetLocalIP returned empty string")
	}

	// Should be a valid IP format (basic check)
	if len(ip) < 7 { // minimum "1.1.1.1"
		t.Errorf("GetLocalIP returned invalid IP: %s", ip)
	}
}

func TestAddressFromNetAddr(t *testing.T) {
	// Test with nil
	addr := network.AddressFromNetAddr(nil)
	if addr.IP != "" || addr.Port != 0 {
		t.Errorf("Expected empty address for nil input, got %v", addr)
	}

	// Test with UDP listener
	net := network.NewUDPNetwork()
	listener, err := net.Listen(Address{IP: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	addr = network.AddressFromNetAddr(listener.LocalAddr())
	if addr.IP != "127.0.0.1" {
		t.Errorf("Expected IP 127.0.0.1, got %s", addr.IP)
	}
	if addr.Port == 0 {
		t.Error("Expected non-zero port")
	}
}

func TestUDPConnectionClose(t *testing.T) {
	net := network.NewUDPNetwork()

	addr := Address{IP: "127.0.0.1", Port: 0}
	conn, err := net.Listen(addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	// Test Close
	if err := conn.Close(); err != nil {
		t.Errorf("Failed to close connection: %v", err)
	}

	// Test Close on nil connection - this test needs to be more careful
	// We can't directly test a nil *UDPConnection, so we'll skip this part
}

func TestConnectionTimeout(t *testing.T) {
	net := network.NewUDPNetwork()

	// Try to dial a non-existent address
	addr := Address{IP: "192.0.2.1", Port: 12345} // TEST-NET-1 (RFC 5737)
	conn, err := net.Dial(addr)
	if err != nil {
		t.Skip("Dial failed (expected in some environments)")
	}
	defer conn.Close()

	// Try to receive (should timeout)
	start := time.Now()
	_, err = conn.Recv()
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	// Should timeout within reasonable time (deadline is set to 5 seconds)
	if duration > 10*time.Second {
		t.Errorf("Timeout took too long: %v", duration)
	}
}

func TestWireMessageEncoding(t *testing.T) {
	// Test the wire message encoding/decoding indirectly through Send/Recv
	net := network.NewUDPNetwork()

	addr := Address{IP: "127.0.0.1", Port: 0}
	listener, err := net.Listen(addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	listenerAddr := network.AddressFromNetAddr(listener.LocalAddr())

	// Test with complex contact data - use correct Triple structure
	testContact := Triple{
		ID:   []byte{255, 254, 253, 252, 251, 250, 249, 248, 247, 246, 245, 244, 243, 242, 241, 240, 239, 238, 237, 236}, // 20 bytes
		Addr: Address{IP: "192.168.1.100", Port: 8080},
		Port: 8080,
	}

	// Test with binary payload
	testPayload := []byte{0, 1, 2, 3, 255, 254, 253}

	msg := network.Message{
		FromContact: testContact,
		To:          listenerAddr,
		Payload:     testPayload,
		Network:     net,
	}

	// Send message via connection
	conn, err := net.Dial(listenerAddr)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	go func() {
		if err := conn.Send(msg); err != nil {
			t.Errorf("Failed to send message: %v", err)
		}
	}()

	// Receive and verify
	received, err := listener.Recv()
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	if !bytes.Equal(received.Payload, testPayload) {
		t.Errorf("Payload mismatch after encoding/decoding")
	}

	if !bytes.Equal(received.FromContact.ID, testContact.ID) {
		t.Errorf("ID mismatch after encoding/decoding")
	}
}

// Helper function to create a 20-byte ID
func createTestID(pattern byte) []byte {
	id := make([]byte, 20)
	for i := range id {
		id[i] = pattern
	}
	return id
}

// Additional test for edge cases
func TestMessageWithEmptyPayload(t *testing.T) {
	net := network.NewUDPNetwork()

	addr := Address{IP: "127.0.0.1", Port: 0}
	listener, err := net.Listen(addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	listenerAddr := network.AddressFromNetAddr(listener.LocalAddr())

	// Create message with empty payload
	testContact := Triple{
		ID:   createTestID(42),
		Addr: Address{IP: "127.0.0.1", Port: 12345},
		Port: 12345,
	}

	msg := network.Message{
		FromContact: testContact,
		To:          listenerAddr,
		Type:        "test",
		Payload:     []byte{}, // Empty payload
		Network:     net,
	}

	conn, err := net.Dial(listenerAddr)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	go func() {
		if err := conn.Send(msg); err != nil {
			t.Errorf("Failed to send message: %v", err)
		}
	}()

	received, err := listener.Recv()
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	if len(received.Payload) != 0 {
		t.Errorf("Expected empty payload, got %d bytes", len(received.Payload))
	}
}

// Test network partition healing
func TestNetworkPartitionHealing(t *testing.T) {
	net := network.NewUDPNetwork()

	addr1 := Address{IP: "127.0.0.1", Port: 8001}
	addr2 := Address{IP: "127.0.0.1", Port: 8002}
	addr3 := Address{IP: "127.0.0.1", Port: 8003}

	// Create partition
	net.Partition([]Address{addr1, addr2}, []Address{addr3})

	// Verify partition exists
	_, err1 := net.Dial(addr1)
	_, err2 := net.Dial(addr2)
	_, err3 := net.Dial(addr3)

	if err1 == nil || err2 == nil || err3 == nil {
		t.Error("Expected errors when dialing partitioned addresses")
	}

	// Heal network
	net.Heal()

	// After healing, the partition should be gone (though connections may still fail due to no listeners)
	// This is mainly testing that the partition state is cleared
}
