package cli

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

		fmt.Printf("Uploading content...\n")
		// TODO: Implement actual upload to Kademlia network
		// For now, just simulate successful upload
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
		// TODO: Implement actual lookup in Kademlia network
		// For now, just simulate retrieval
		fmt.Printf("Object not found in network\n")
		// When implemented, should show:
		// fmt.Printf("Content: %s\nRetrieved from node: %s\n", content, nodeAddress)
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
