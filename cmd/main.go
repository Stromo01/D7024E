package main

import (
	"log"

	"github.com/eislab-cps/go-template/internal/cli"
	"github.com/eislab-cps/go-template/pkg/build"
	"github.com/eislab-cps/go-template/pkg/kademlia"
)

var (
	BuildVersion string = ""
	BuildTime    string = ""
)

func main() {
	build.BuildVersion = BuildVersion
	build.BuildTime = BuildTime

	network := "tcp"         // or the appropriate network type, e.g., "udp"
	addr := "localhost:8000" // or the appropriate address

	node, err := kademlia.NewNode(network, addr)
	if err != nil {
		log.Fatal(err)
	}
	cli.SetNode(node)
	cli.Execute()
}
