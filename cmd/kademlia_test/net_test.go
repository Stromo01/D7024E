package kademlia_test

import (
	"testing"

	"github.com/Stromo01/D7024E/cmd/kademlia"
)

func TestNetBasic(t *testing.T) {
	// TODO: implement
}

func TestMessageStruct(t *testing.T) {
	addr := kademlia.Address{IP: "127.0.0.1", Port: 8000}
	msg := kademlia.Message{From: addr, To: addr, Payload: []byte("hello")}
	if msg.From != addr || msg.To != addr || string(msg.Payload) != "hello" {
		t.Errorf("Message struct fields not set correctly")
	}
}

func TestNetworkInterface(t *testing.T) {
	net := &dummyNetwork{}
	_, err := net.Listen(kademlia.Address{IP: "127.0.0.1", Port: 8000})
	if err != nil {
		t.Errorf("Listen should not error in dummyNetwork")
	}
	_, err = net.Dial(kademlia.Address{IP: "127.0.0.1", Port: 8000})
	if err != nil {
		t.Errorf("Dial should not error in dummyNetwork")
	}
}
