package main

import (
	"fmt"
	"os"
	"sync"
	"time"

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

	var wg sync.WaitGroup
	tNodeAlive := 10 * time.Minute // Set your desired lifetime

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Node main loop: process messages, etc.
		aliveTimer := time.NewTimer(tNodeAlive)
		for {
			select {
			case <-aliveTimer.C:
				fmt.Println("Node lifetime expired, shutting down.")
				return
				// Add other cases here for message handling, etc.
			}
		}
	}()

	wg.Wait()
	fmt.Println("Node exited.")
}
