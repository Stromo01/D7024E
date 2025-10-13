package cli

import (
	"crypto/sha1"
	"fmt"
)

func HandlePut(data string) {
	if currentNode == nil {
		fmt.Println("Error: No node available")
		return
	}

	hash := fmt.Sprintf("%x", sha1.Sum([]byte(data)))

	if node, ok := currentNode.(interface{ IterativeStore(string, []byte) }); ok {
		node.IterativeStore(hash, []byte(data))
		fmt.Printf("%s\n", hash)
	} else {
		fmt.Println("Error: Node does not support IterativeStore method")
	}
}
