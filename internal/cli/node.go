package cli

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/eislab-cps/go-template/pkg/kademlia"
	"github.com/spf13/cobra"
)

// Global node instance
var currentNode *kademlia.Node

// SetNode sets the current node instance
func SetNode(node *kademlia.Node) {
	currentNode = node
}

func init() {
	rootCmd.AddCommand(nodeCmd)
	nodeCmd.AddCommand(putCmd)
	nodeCmd.AddCommand(getCmd)
	nodeCmd.AddCommand(exitCmd)
}

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Kademlia node operations",
	Long:  "Commands for managing Kademlia nodes",
}

var putCmd = &cobra.Command{
	Use:   "put [file contents]",
	Short: "Upload file contents and get hash",
	Long:  "Takes the contents of the file you are uploading and outputs the hash of the object if it can be uploaded successfully",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contents := args[0]

		// Generate hash of the contents
		hash := sha1.New()
		hash.Write([]byte(contents))
		hashBytes := hash.Sum(nil)
		hashString := fmt.Sprintf("%x", hashBytes)

		if currentNode == nil {
			fmt.Printf("Error: No node running. Start a node first.\n")
			return
		}

		fmt.Printf("Uploading content...\n")

		// Store the object in the current node
		currentNode.StoreObject(hashString, []byte(contents))

		fmt.Printf("Successfully uploaded. Hash: %s\n", hashString)
	},
}

var getCmd = &cobra.Command{
	Use:   "get [hash]",
	Short: "Download object by hash",
	Long:  "Takes a hash as its only argument and outputs the contents of the object and the node it was retrieved from if it could be downloaded successfully",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hash := args[0]

		fmt.Printf("Searching for object with hash: %s\n", hash)

		if currentNode == nil {
			fmt.Printf("Error: No node running. Start a node first.\n")
			return
		}

		// First check if we have the object locally
		if value, found := currentNode.FindObject(hash); found {
			fmt.Printf("Content: %s\nRetrieved from local node: %s\n", string(value), currentNode.Address().String())
			return
		}

		// If not found locally, search the network
		nodes := currentNode.IterativeFindNode(hash)
		if len(nodes) == 0 {
			fmt.Printf("Object not found in network\n")
		} else {
			fmt.Printf("Found %d potential nodes, but value retrieval not yet implemented\n", len(nodes))
		}
	},
}

var exitCmd = &cobra.Command{
	Use:   "exit",
	Short: "Terminate the node",
	Long:  "Terminates the node and exits the program",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Terminating node...")
		// TODO: Implement proper node shutdown
		// Should close connections, save state, etc.
		os.Exit(0)
	},
}
