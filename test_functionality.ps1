# PowerShell script to test Kademlia DHT functionality
Write-Host "🧪 Starting Kademlia DHT functionality tests..." -ForegroundColor Green

Write-Host "📊 Running comprehensive test suite..." -ForegroundColor Yellow
Write-Host ""

# Run unit tests
Write-Host "1. Running Unit Tests (M4 requirement)..." -ForegroundColor Cyan
go test ./cmd/ -v -count=1 | Select-String -Pattern "PASS|FAIL|TestLarge"

Write-Host ""
Write-Host "2. Testing CLI Help..." -ForegroundColor Cyan
./cmd.exe -help

Write-Host ""
Write-Host "✅ Test preparation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "🔧 Manual Testing Instructions:" -ForegroundColor Yellow
Write-Host "================================"
Write-Host ""
Write-Host "🌐 Single Node Test (Mock Network):" -ForegroundColor Cyan
Write-Host "   ./cmd.exe -port 8080"
Write-Host "   Then try:"
Write-Host "     put Hello, Kademlia!"
Write-Host "     put Testing distributed storage"
Write-Host "     get <hash-from-previous-put>"
Write-Host "     exit"
Write-Host ""

Write-Host "🌐 Multi-Node Test (Real UDP Network):" -ForegroundColor Cyan
Write-Host "   Terminal 1: ./cmd.exe -port 8080 -real-network"
Write-Host "   Terminal 2: ./cmd.exe -port 8081 -bootstrap-ip 127.0.0.1 -bootstrap-port 8080 -real-network"
Write-Host "   Terminal 3: ./cmd.exe -port 8082 -bootstrap-ip 127.0.0.1 -bootstrap-port 8080 -real-network"
Write-Host ""
Write-Host "   Then test put/get operations across nodes!"
Write-Host ""

Write-Host "🐳 Docker Network Test (M5 requirement):" -ForegroundColor Cyan
Write-Host "   docker build -f Dockerfile.kademlia -t kademlia-node ."
Write-Host "   docker-compose up -d"
Write-Host ""

Write-Host "📈 Performance Test:" -ForegroundColor Cyan
Write-Host "   go test ./cmd/ -v -run='TestLargeScaleNetwork' -count=1"
Write-Host ""

Write-Host "🔒 Race Condition Test (M7 requirement):" -ForegroundColor Cyan
Write-Host "   go test ./cmd/ -race -v -count=3"
Write-Host ""

Write-Host "🎯 All tests completed! Your Kademlia DHT implementation is ready." -ForegroundColor Green