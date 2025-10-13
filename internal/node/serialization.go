package node

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	. "github.com/eislab-cps/go-template/pkg/kademlia"
)

func tripleDeserialize(data string) ([]Triple, error) {
	if data == "" {
		return []Triple{}, nil
	}

	parts := strings.Split(data, ",")
	var triples []Triple

	for _, part := range parts {
		components := strings.Split(part, ":")
		if len(components) != 3 {
			continue // Skip malformed entries
		}

		ip := components[0]
		port, err := strconv.Atoi(components[1])
		if err != nil {
			continue // Skip malformed entries
		}

		id, err := hex.DecodeString(components[2])
		if err != nil {
			continue // Skip malformed entries
		}

		triple := Triple{
			ID:   id,
			Addr: Address{IP: ip, Port: port},
			Port: port,
		}
		triples = append(triples, triple)
	}

	return triples, nil
}

func tripleSerialize(triples []Triple) string {
	var parts []string
	for _, triple := range triples {
		part := fmt.Sprintf("%s:%d:%x",
			triple.Addr.IP,
			triple.Addr.Port,
			triple.ID)
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
}
