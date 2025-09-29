#!/bin/bash

# Script to generate docker-compose.yml for Kademlia DHT network with 50+ nodes

NODE_COUNT=50
COMPOSE_FILE="docker-compose-generated.yml"

cat > $COMPOSE_FILE << 'EOF'
version: '3.8'

services:
  # Bootstrap node (first node in the network)
  kademlia-bootstrap:
    build:
      context: .
      dockerfile: Dockerfile.kademlia
    container_name: kademlia-bootstrap
    ports:
      - "8080:8080"
    command: ["./kademlia", "node", "--port", "8080"]
    networks:
      kademlia-net:
        ipv4_address: 172.20.0.10
    environment:
      - NODE_ID=bootstrap
    healthcheck:
      test: ["CMD-SHELL", "nc -z localhost 8080 || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3

EOF

# Generate additional nodes
for i in $(seq 1 $NODE_COUNT); do
  cat >> $COMPOSE_FILE << EOF
  kademlia-node-$i:
    build:
      context: .
      dockerfile: Dockerfile.kademlia
    depends_on:
      - kademlia-bootstrap
    command: ["./kademlia", "node", "--port", "8080", "--bootstrap-ip", "172.20.0.10", "--bootstrap-port", "8080"]
    networks:
      - kademlia-net
    environment:
      - NODE_ID=node-$i
    restart: unless-stopped

EOF
done

# Add networks section
cat >> $COMPOSE_FILE << 'EOF'
networks:
  kademlia-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
EOF

echo "Generated docker-compose file with $(($NODE_COUNT + 1)) nodes: $COMPOSE_FILE"
echo ""
echo "To start the network:"
echo "  docker-compose -f $COMPOSE_FILE up -d"
echo ""
echo "To stop the network:"
echo "  docker-compose -f $COMPOSE_FILE down"
echo ""
echo "To view logs from all nodes:"
echo "  docker-compose -f $COMPOSE_FILE logs -f"
echo ""
echo "To view logs from specific node:"
echo "  docker-compose -f $COMPOSE_FILE logs -f kademlia-node-1"