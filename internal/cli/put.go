package cli

import (
	"crypto/sha1"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:   "put [data]",
	Short: "Store data in a node",
	Long:  "Store data in a Kademlia node and return the corresponding hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data := args[0]
		hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))

		// Get the port from environment (same as the running node)
		port := "8000" // default
		if nodePort := os.Getenv("NODE_PORT"); nodePort != "" {
			port = nodePort
		}

		nodeAddr := fmt.Sprintf("localhost:%s", port)
		err := sendStoreMessage(nodeAddr, hash, data)
		if err != nil {
			fmt.Printf("Error storing data: %v\n", err)
			return
		}

		fmt.Printf("%s\n", hash)
	},
}

func sendStoreMessage(nodeAddr, key, value string) error {
	// Create UDP connection
	conn, err := net.Dial("udp", nodeAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to node: %v", err)
	}
	defer conn.Close()

	// Set timeout
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Create message in the format your node expects
	// Check your node's message handling to see the exact format needed
	message := fmt.Sprintf(`{"type":"STORE","key":"%s","value":"%s"}`, key, value)

	// Send the message
	_, err = conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	// Read response (optional)
	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer)
	if err != nil {
		// Don't fail if no response, just log
		fmt.Printf("Note: No response from node (this might be expected)\n")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(putCmd)
}
