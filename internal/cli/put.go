package cli

import (
	"crypto/sha1"
	"fmt"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:   "put [data]",
	Short: "Store data in a node",
	Long:  "Store data in a Kademlia node and return the corresponding hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if currentNode == nil {
			fmt.Println("Error: No node available")
			return
		}
		data := args[0]
		hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))

		if node, ok := currentNode.(interface{ iterativeStore(string, []byte) }); ok {
			node.iterativeStore(hash, []byte(data))
			fmt.Printf("%s\n", hash)
		} else {
			fmt.Println("Error: Node does not support iterativeStore method")
		}

		fmt.Printf("%s\n", hash)
	},
}

func init() {
	rootCmd.AddCommand(putCmd)
}
