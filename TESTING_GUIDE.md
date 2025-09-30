# 🧪 Kademlia DHT Testing Guide

## 🎯 **Testing Strategy Overview**

This guide shows you how to thoroughly test your Kademlia DHT implementation across all mandatory requirements M1-M7.

## 📋 **1. Unit Tests (M4 Requirement)**

### **Run All Tests**
```bash
# Run all tests with verbose output
go test ./cmd/ -v -count=1

# Run with race detection (M7 requirement)
go test ./cmd/ -v -race -count=1

# Run specific test categories
go test ./cmd/ -v -run="TestBasic"       # Basic functionality
go test ./cmd/ -v -run="TestIterative"  # Iterative operations  
go test ./cmd/ -v -run="TestLargeScale" # Large scale networks
go test ./cmd/ -v -run="TestRPC"        # RPC operations
```

### **Test Coverage Analysis**
```bash
# Generate test coverage report
go test ./cmd/ -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View coverage in terminal
go test ./cmd/ -cover
```

### **Key Tests to Verify**
- ✅ **TestLargeScaleNetwork**: 1000+ nodes with packet dropping
- ✅ **TestIterativeFindValue**: Network-wide value retrieval
- ✅ **TestIterativeStore**: K-way replication storage
- ✅ **TestRPCResponseMatching**: Async RPC system
- ✅ **TestConfigurableNetwork**: Various network sizes

## 🖥️ **2. Manual CLI Testing (M3 Requirement)**

### **Test 1: Single Node Operations**
```bash
# Build the application
go build ./cmd

# Start a single node (mock network for testing)
./cmd.exe -port 8080

# Test CLI commands:
kademlia> put "Hello, Kademlia DHT!"
📦 Storing data: 'Hello, Kademlia DHT!'
✅ Data stored successfully!
🔑 Hash: a1b2c3d4e5f6789012345678901234567890abcd

kademlia> get a1b2c3d4e5f6789012345678901234567890abcd
🔍 Searching for data with hash: a1b2c3d4e5f6789012345678901234567890abcd
✅ Content: Hello, Kademlia DHT!
📍 Retrieved from network (searched 1 nodes)

kademlia> exit
👋 Exiting...
```

### **Test 2: Multi-Node Network (Real UDP)**

**Terminal 1 - Bootstrap Node:**
```bash
./cmd.exe -port 8080 -real-network
# Wait for "Node started successfully!" message
```

**Terminal 2 - Second Node:**
```bash
./cmd.exe -port 8081 -bootstrap-ip 127.0.0.1 -bootstrap-port 8080 -real-network
# Should show "✅ Successfully connected to bootstrap node"
```

**Terminal 3 - Third Node:**
```bash
./cmd.exe -port 8082 -bootstrap-ip 127.0.0.1 -bootstrap-port 8080 -real-network
```

**Test Network Operations:**
```bash
# In Terminal 1 (port 8080):
kademlia> put "Distributed data test"
🔑 Hash: def789abc123456789012345678901234567890

# In Terminal 2 (port 8081):
kademlia> get def789abc123456789012345678901234567890
✅ Content: Distributed data test
📍 Retrieved from network (searched 3 nodes)

# In Terminal 3 (port 8082):
kademlia> put "Another test message"
🔑 Hash: 987654321098765432109876543210987654321a

# In Terminal 1:
kademlia> get 987654321098765432109876543210987654321a
✅ Content: Another test message
```

## 🐳 **3. Docker Container Testing (M5 Requirement)**

### **Test 1: Small Network (5 nodes)**
```bash
# Build Docker image
docker build -f Dockerfile.kademlia -t kademlia-node .

# Test with small network
docker-compose up -d kademlia-bootstrap kademlia-node-1 kademlia-node-2

# Check logs
docker-compose logs -f kademlia-bootstrap
docker-compose logs -f kademlia-node-1

# Connect to a node for testing
docker exec -it kademlia-bootstrap /bin/sh
# Inside container: ./kademlia -port 8080 -real-network
```

### **Test 2: Large Network (50+ nodes)**
```bash
# Generate 50-node network
.\generate-docker-compose.ps1 -NodeCount 50

# Start the entire network
docker-compose -f docker-compose-generated.yml up -d

# Monitor network formation
docker-compose -f docker-compose-generated.yml logs -f | grep "Successfully connected"

# Check network health
docker-compose -f docker-compose-generated.yml ps

# Clean up
docker-compose -f docker-compose-generated.yml down
```

## 🔬 **4. Network Formation Testing (M1 Requirement)**

### **Test PING Operations**
```bash
# Create test script
cat > test_ping.go << 'EOF'
package main

import (
    "fmt"
    "time"
)

func testPing() {
    network := NewMockNetwork()
    
    nodeA, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8001})
    nodeB, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8002})
    
    nodeA.Start()
    nodeB.Start()
    
    time.Sleep(100 * time.Millisecond)
    
    err := nodeA.RPCPing(nodeB.addr)
    if err != nil {
        fmt.Printf("❌ PING failed: %v\n", err)
    } else {
        fmt.Printf("✅ PING successful\n")
    }
    
    nodeA.Close()
    nodeB.Close()
}

func main() {
    testPing()
}
EOF

go run test_ping.go
```

### **Test Bootstrap Joining**
```bash
# Test script for network joining
go test ./cmd/ -v -run="TestNodeCommunication"
go test ./cmd/ -v -run="TestIterativeFindNode"
```

## 📊 **5. Object Distribution Testing (M2 Requirement)**

### **Test K-way Replication**
```bash
# Run replication test
go test ./cmd/ -v -run="TestIterativeStore"

# Manual replication test
cat > test_replication.go << 'EOF'
package main

import (
    "fmt"
)

func testReplication() {
    network := NewMockNetwork()
    
    // Create multiple nodes
    nodes := make([]*Node, 5)
    for i := 0; i < 5; i++ {
        addr := Address{IP: "127.0.0.1", Port: 8000 + i}
        node, _ := NewNode(network, addr)
        nodes[i] = node
        node.Start()
        
        // Add other nodes to routing table
        for j := 0; j < i; j++ {
            contact := Triple{
                ID:   nodes[j].id[:],
                Addr: nodes[j].addr,
                Port: nodes[j].addr.Port,
            }
            node.routing.addContact(contact)
            nodes[j].routing.addContact(Triple{
                ID:   node.id[:],
                Addr: node.addr,
                Port: node.addr.Port,
            })
        }
    }
    
    // Test storing and finding
    key := "test-replication-key"
    value := []byte("test-replication-value")
    
    err := nodes[0].IterativeStore(key, value)
    if err != nil {
        fmt.Printf("❌ Store failed: %v\n", err)
        return
    }
    
    // Try to find from different node
    foundValue, contacts, found := nodes[4].IterativeFindValue(key)
    if found {
        fmt.Printf("✅ Replication successful: %s\n", string(foundValue))
        fmt.Printf("📍 Found via %d contacts\n", len(contacts))
    } else {
        fmt.Printf("❌ Replication failed\n")
    }
    
    // Clean up
    for _, node := range nodes {
        node.Close()
    }
}

func main() {
    testReplication()
}
EOF

go run test_replication.go
```

## ⚡ **6. Concurrency Testing (M7 Requirement)**

### **Race Condition Testing**
```bash
# Run all tests with race detection
go test ./cmd/ -race -v -count=5

# Specific race condition tests
go test ./cmd/ -race -v -run="TestRPCChannelRace"
go test ./cmd/ -race -v -run="TestConcurrent"
```

### **Load Testing**
```bash
# Large scale concurrent operations
go test ./cmd/ -v -run="TestLargeScaleNetwork" -timeout=30s

# Benchmark performance
go test ./cmd/ -bench=. -benchmem
```

## 🔧 **7. Fault Tolerance Testing**

### **Packet Dropping Tests**
```bash
# Test various packet drop rates
go test ./cmd/ -v -run="TestConfigurableNetwork"

# Manual packet dropping test
cat > test_packet_drop.go << 'EOF'
package main

import (
    "fmt"
)

func testPacketDrop() {
    // Test with 0%, 25%, 50%, 75% packet drop
    dropRates := []float64{0.0, 0.25, 0.5, 0.75}
    
    for _, rate := range dropRates {
        fmt.Printf("\n🧪 Testing with %.0f%% packet drop rate\n", rate*100)
        
        network := NewLargeScaleNetwork(rate)
        
        nodeA, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8001})
        nodeB, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8002})
        
        nodeA.Start()
        nodeB.Start()
        
        // Store data
        key := fmt.Sprintf("test-key-%.0f", rate*100)
        value := []byte(fmt.Sprintf("test-value-%.0f", rate*100))
        
        err := nodeA.IterativeStore(key, value)
        if err != nil {
            fmt.Printf("❌ Store failed with %.0f%% drop rate: %v\n", rate*100, err)
        } else {
            fmt.Printf("✅ Store succeeded with %.0f%% drop rate\n", rate*100)
        }
        
        nodeA.Close()
        nodeB.Close()
    }
}

func main() {
    testPacketDrop()
}
EOF

go run test_packet_drop.go
```

## 📈 **8. Performance Testing**

### **Network Size Scaling**
```bash
# Test different network sizes
go test ./cmd/ -v -run="TestLargeScaleNetwork" -args -nodes=100
go test ./cmd/ -v -run="TestLargeScaleNetwork" -args -nodes=500
go test ./cmd/ -v -run="TestLargeScaleNetwork" -args -nodes=1000
```

### **Throughput Testing**
```bash
# Create throughput test
cat > test_throughput.go << 'EOF'
package main

import (
    "fmt"
    "time"
)

func testThroughput() {
    network := NewMockNetwork()
    
    node, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8080})
    node.Start()
    defer node.Close()
    
    // Store multiple objects
    numObjects := 100
    start := time.Now()
    
    for i := 0; i < numObjects; i++ {
        key := fmt.Sprintf("key-%d", i)
        value := []byte(fmt.Sprintf("value-%d", i))
        node.IterativeStore(key, value)
    }
    
    duration := time.Since(start)
    rate := float64(numObjects) / duration.Seconds()
    
    fmt.Printf("✅ Stored %d objects in %v\n", numObjects, duration)
    fmt.Printf("📊 Throughput: %.2f operations/second\n", rate)
}

func main() {
    testThroughput()
}
EOF

go run test_throughput.go
```

## 🎯 **9. Integration Testing Checklist**

### **Complete M1-M7 Validation**
```bash
# 1. M1: Network Formation
./cmd.exe -port 8080 -real-network &
./cmd.exe -port 8081 -bootstrap-ip 127.0.0.1 -bootstrap-port 8080 -real-network &

# 2. M2: Object Distribution
# Test put/get across nodes (shown above)

# 3. M3: CLI Interface  
# Test all CLI commands (shown above)

# 4. M4: Unit Testing
go test ./cmd/ -v -cover -race

# 5. M5: Containerization
docker-compose up -d

# 7. M7: Concurrency
go test ./cmd/ -race -v -count=10
```

## 🏆 **Expected Test Results**

### **Success Indicators**
- ✅ All unit tests pass (40+ tests)
- ✅ CLI commands work across multiple nodes
- ✅ Docker containers can communicate
- ✅ No race conditions detected
- ✅ >50% test coverage achieved
- ✅ Network scales to 1000+ nodes
- ✅ Fault tolerance with packet dropping

### **Troubleshooting Common Issues**
```bash
# If tests fail:
go mod tidy              # Fix dependencies
go clean -testcache      # Clear test cache
go test ./cmd/ -v -count=1  # Force fresh run

# If networking fails:
netstat -an | grep 8080  # Check port usage
ss -tulpn | grep 8080    # Linux alternative
```

This comprehensive testing approach validates all mandatory requirements M1-M7!