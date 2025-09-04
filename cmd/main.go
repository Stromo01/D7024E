package main

import (
	"fmt"
	"os"

	"github.com/Stromo01/D7024E/pkg/build"
)

var (
	BuildVersion string = ""
	BuildTime    string = ""
)

func main() {
	build.BuildVersion = BuildVersion
	build.BuildTime = BuildTime

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID, _ = os.Hostname()
	}
	nodePort := os.Getenv("NODE_PORT")
	fmt.Println("Node ID:", nodeID)
	fmt.Println("Node Port:", nodePort)

	//cli.Execute()
}
