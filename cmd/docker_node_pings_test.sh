#!/bin/bash
docker compose up --scale kademliaNode=50 -d
# List all kademliaNode containers
nodes=$(docker ps --format '{{.Names}}' | grep d7024e-kademliaNode)

echo "Testing pings between all running Docker nodes..."

for src in $nodes; do
  for dst in $nodes; do
    if [ "$src" != "$dst" ]; then
      echo "Ping from $src to $dst:"
      docker exec "$src" ping -c 1 "$dst"
      echo ""
    fi
  done
done

echo "All pings complete."