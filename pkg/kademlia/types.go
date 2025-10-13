package kademlia

import "fmt"

type Address struct {
	IP   string
	Port int
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

type Triple struct {
	ID   []byte
	Addr Address
	Port int
}

func (t Triple) String() string {
	return fmt.Sprintf("{ID: %x, Addr: %s, Port: %d}", t.ID, t.Addr.String(), t.Port)
}
