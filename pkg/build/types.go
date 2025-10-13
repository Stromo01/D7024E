package kademlia

type Address struct {
	IP   string
	Port int
}

type Triple struct {
	ID   []byte
	Addr Address
	Port int
}
