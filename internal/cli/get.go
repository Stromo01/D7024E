package cli

import (
	"fmt"
)

func HandleGet(hash string) {
	if CurrentNode == nil {
		fmt.Println("Error: No node available")
		return
	}

	if node, ok := CurrentNode.(interface {
		FindObject(string) ([]byte, string, bool)
	}); ok {
		data, triple, found := node.FindObject(hash)
		if found {
			fmt.Printf("Data: %s\nTriple: %s\n", string(data), triple) //TODO: Use triple struct?
		} else {
			fmt.Println("Error: Data not found")
		}
		fmt.Printf("%s\n", hash)
	} else {
		fmt.Println("Error: Node does not support IterativeStore method")
	}
}
