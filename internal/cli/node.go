package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(nodeCmd)
	nodeCmd.AddCommand(startCmd)
	nodeCmd.AddCommand(joinCmd)
	nodeCmd.AddCommand(storeCmd)
	nodeCmd.AddCommand(findCmd)

	// Add flags
	startCmd.Flags().StringP("address", "a", "localhost:8080", "Node address (IP:Port)")
	joinCmd.Flags().StringP("address", "a", "localhost:8080", "This node's address")
	joinCmd.Flags().StringP("bootstrap", "b", "", "Bootstrap node address to join")
	storeCmd.Flags().StringP("address", "a", "localhost:8080", "Node address")
	storeCmd.Flags().StringP("key", "k", "", "Key to store")
	storeCmd.Flags().StringP("value", "v", "", "Value to store")
	findCmd.Flags().StringP("address", "a", "localhost:8080", "Node address")
	findCmd.Flags().StringP("key", "k", "", "Key to find")
}

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Kademlia node operations",
	Long:  "Commands for managing Kademlia nodes",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a Kademlia node",
	Long:  "Start a new Kademlia node on the specified address",
	Run: func(cmd *cobra.Command, args []string) {
		address, _ := cmd.Flags().GetString("address")
		fmt.Printf("Starting Kademlia node on %s\n", address)
		// TODO: Implement node start logic
		// You would create and start your node here
	},
}

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join a Kademlia network",
	Long:  "Start a node and join an existing Kademlia network",
	Run: func(cmd *cobra.Command, args []string) {
		address, _ := cmd.Flags().GetString("address")
		bootstrap, _ := cmd.Flags().GetString("bootstrap")

		if bootstrap == "" {
			fmt.Println("Error: bootstrap node address is required")
			return
		}

		fmt.Printf("Starting node on %s and joining network via %s\n", address, bootstrap)
		// TODO: Implement join network logic
	},
}

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Store a key-value pair",
	Long:  "Store a key-value pair in the Kademlia network",
	Run: func(cmd *cobra.Command, args []string) {
		address, _ := cmd.Flags().GetString("address")
		key, _ := cmd.Flags().GetString("key")
		value, _ := cmd.Flags().GetString("value")

		if key == "" || value == "" {
			fmt.Println("Error: both key and value are required")
			return
		}

		fmt.Printf("Storing %s=%s from node %s\n", key, value, address)
		// TODO: Implement store logic
	},
}

var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Find a value by key",
	Long:  "Find a value by key in the Kademlia network",
	Run: func(cmd *cobra.Command, args []string) {
		address, _ := cmd.Flags().GetString("address")
		key, _ := cmd.Flags().GetString("key")

		if key == "" {
			fmt.Println("Error: key is required")
			return
		}

		fmt.Printf("Finding key %s from node %s\n", key, address)
		// TODO: Implement find logic
	},
}
