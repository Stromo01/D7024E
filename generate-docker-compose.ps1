# PowerShell script to generate docker-compose.yml for Kademlia DHT network with 50+ nodes

param(
    [int]$NodeCount = 50
)

$ComposeFile = "docker-compose-generated.yml"

# Create the header
$Header = @"
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
    command: ["./kademlia", "-port", "8080", "-real-network"]
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

"@

$Header | Out-File -FilePath $ComposeFile -Encoding UTF8

# Generate additional nodes
for ($i = 1; $i -le $NodeCount; $i++) {
    $NodeConfig = @"
  kademlia-node-$i:
    build:
      context: .
      dockerfile: Dockerfile.kademlia
    depends_on:
      - kademlia-bootstrap
    command: ["./kademlia", "-port", "8080", "-bootstrap-ip", "172.20.0.10", "-bootstrap-port", "8080", "-real-network"]
    networks:
      - kademlia-net
    environment:
      - NODE_ID=node-$i
    restart: unless-stopped

"@
    $NodeConfig | Out-File -FilePath $ComposeFile -Append -Encoding UTF8
}

# Add networks section
$Networks = @"
networks:
  kademlia-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
"@

$Networks | Out-File -FilePath $ComposeFile -Append -Encoding UTF8

Write-Host "Generated docker-compose file with $($NodeCount + 1) nodes: $ComposeFile"
Write-Host ""
Write-Host "To start the network:"
Write-Host "  docker-compose -f $ComposeFile up -d"
Write-Host ""
Write-Host "To stop the network:"
Write-Host "  docker-compose -f $ComposeFile down"
Write-Host ""
Write-Host "To view logs from all nodes:"
Write-Host "  docker-compose -f $ComposeFile logs -f"
Write-Host ""
Write-Host "To view logs from specific node:"
Write-Host "  docker-compose -f $ComposeFile logs -f kademlia-node-1"
Write-Host ""
Write-Host "To scale to different number of nodes:"
Write-Host "  .\generate-docker-compose.ps1 -NodeCount 100"