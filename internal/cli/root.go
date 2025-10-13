package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Verbose bool
var currentNode interface{}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}

var rootCmd = &cobra.Command{
	Use:   "kademlia", // Changed from "helloworld"
	Short: "Kademlia DHT implementation",
	Long:  "A Kademlia Distributed Hash Table implementation",
}

func Execute(node interface{}) {
	currentNode = node
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
