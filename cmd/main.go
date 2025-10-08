package main

import (
	"log"
	"net"
	"strconv"

	"github.com/eislab-cps/go-template/internal/cli"
	"github.com/eislab-cps/go-template/pkg/build"
	"github.com/eislab-cps/go-template/pkg/kademlia"
)

var (
	BuildVersion string = ""
	BuildTime    string = ""
)
var globalNetwork kademlia.Network

func main() {
	build.BuildVersion = BuildVersion
	build.BuildTime = BuildTime
	globalNetwork = NewMockNetwork()
	IP, err := getLocalIP()
	if err != nil {
		log.Fatal(err)
	}
	PortStr, err := getPortFromAddress(IP)
	if err != nil {
		log.Fatal(err)
	}
	Port, err := strconv.Atoi(PortStr)
	if err != nil {
		log.Fatal(err)
	}
	addr := kademlia.Address{
		IP:   IP,
		Port: Port,
	}
	node, err := kademlia.NewNode(globalNetwork, addr)
	if err != nil {
		log.Fatal(err)
	}
	cli.SetNode(node)
	cli.Execute()
}

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func getPortFromAddress(address string) (string, error) {
	_, port, err := net.SplitHostPort(address)
	return port, err
}
