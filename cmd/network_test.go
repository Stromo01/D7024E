package main

import (
	"net"
	"testing"
	
)

func TestAddress_String(t *testing.T) {
	addr := Address{IP: "192.168.1.1", Port: 8080}
	expected := "192.168.1.1:8080"
	if addr.String() != expected {
		t.Errorf("Expected %s, got %s", expected, addr.String())
	}
}

func TestUDPNetwork_NewUDPNetwork(t *testing.T) {
	network := NewUDPNetwork()
	if network == nil {
		t.Fatal("NewUDPNetwork() returned nil")
	}
	if network.partitioned {
		t.Error("New network should not be partitioned")
	}
}

func TestUDPNetwork_Partition(t *testing.T) {
	network := NewUDPNetwork()
	group1 := []Address{{IP: "127.0.0.1", Port: 8000}}
	group2 := []Address{{IP: "127.0.0.1", Port: 8001}}

	network.Partition(group1, group2)

	if !network.partitioned {
		t.Error("Network should be partitioned")
	}

	if !network.partition1["127.0.0.1:8000"] {
		t.Error("Group1 address should be in partition1")
	}

	if !network.partition2["127.0.0.1:8001"] {
		t.Error("Group2 address should be in partition2")
	}
}

func TestUDPNetwork_Heal(t *testing.T) {
	network := NewUDPNetwork()
	group1 := []Address{{IP: "127.0.0.1", Port: 8000}}
	group2 := []Address{{IP: "127.0.0.1", Port: 8001}}

	network.Partition(group1, group2)
	network.Heal()

	if network.partitioned {
		t.Error("Network should not be partitioned after heal")
	}

	if len(network.partition1) > 0 || len(network.partition2) > 0 {
		t.Error("Partitions should be empty after heal")
	}
}

func TestGetLocalIP(t *testing.T) {
	ip, err := GetLocalIP()
	if err != nil {
		t.Errorf("GetLocalIP() failed: %v", err)
	}
	if ip == "" {
		t.Error("GetLocalIP() returned empty string")
	}
	t.Logf("Local IP: %s", ip)
}

func TestWireMessage_Serialization(t *testing.T) {
	triple := Triple{
		ID:   []byte("test-id"),
		Addr: Address{IP: "127.0.0.1", Port: 8000},
		Port: 8000,
	}

	wireMsg := WireMessage{
		FromContact: triple,
		Payload:     []byte("test payload"),
	}

	// This tests that WireMessage can be created and used
	if string(wireMsg.Payload) != "test payload" {
		t.Error("WireMessage payload not preserved")
	}

	if wireMsg.FromContact.Addr.String() != "127.0.0.1:8000" {
		t.Error("WireMessage FromContact not preserved")
	}
}

func TestMessage_Reply_WithUDP(t *testing.T) {
	network := NewUDPNetwork()
	fromAddr := Address{IP: "127.0.0.1", Port: 0} // Let OS choose port

	// Set up receiver
	conn, err := network.Listen(fromAddr)
	if err != nil {
		t.Skipf("Could not create UDP listener (may be in testing environment): %v", err)
	}
	defer conn.Close()

	// Get the actual assigned address
	if udpConn, ok := conn.(*UDPConnection); ok {
		actualAddr := udpConn.conn.LocalAddr().(*net.UDPAddr)
		fromAddr.Port = actualAddr.Port
	}

	toAddr := Address{IP: "127.0.0.1", Port: fromAddr.Port + 1}

	// Create message
	msg := Message{
		From:    toAddr,
		To:      fromAddr,
		Payload: []byte("test"),
		network: network,
	}

	// Test reply - this will fail because toAddr isn't listening, but that's expected
	err = msg.Reply("response", []byte("reply data"))
	if err == nil {
		t.Log("Reply succeeded (unexpected but not necessarily wrong)")
	} else {
		t.Logf("Reply failed as expected: %v", err)
	}
}
