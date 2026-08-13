#!/bin/bash

# Performance Comparison Script: Go vs Python GraphQLmap
# This script benchmarks both implementations against the same GraphQL endpoint

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GRAPHQL_URL="${1:-https://api.github.com/graphql}"
ITERATIONS="${2:-100}"
CONCURRENT="${3:-10}"

echo -e "${YELLOW}=== GraphQLmap Go vs Python Performance Comparison ===${NC}"
echo -e "Target: ${GRAPHQL_URL}"
echo -e "Iterations: ${ITERATIONS}"
echo -e "Concurrent: ${CONCURRENT}"
echo ""

# Check if Go binary exists
if [ ! -f "./graphqlmap-go" ]; then
    echo -e "${YELLOW}Building Go binary...${NC}"
    go build -o graphqlmap-go
fi

# Check if Python version exists
PYTHON_MAP_EXISTS=false
if [ -d "../GraphQLmap" ] && command -v python3 &> /dev/null; then
    PYTHON_MAP_EXISTS=true
    echo -e "${YELLOW}Found Python GraphQLmap${NC}"
fi

echo ""
echo -e "${GREEN}=== Running Go Benchmarks ===${NC}"

# Run Go benchmarks
go test -bench=. -benchmem -count=5 -run=^$ ./... | tee go_benchmarks.txt

echo ""
echo -e "${GREEN}=== Go Performance Summary ===${NC}"
grep -E "^Benchmark" go_benchmarks.txt | awk '{printf "%-40s %10s ops/sec\n", $1, $5}'

if [ "$PYTHON_MAP_EXISTS" = true ]; then
    echo ""
    echo -e "${GREEN}=== Running Python Benchmarks ===${NC}"
    
    # Create Python benchmark script
    cat > python_benchmark.py << 'EOF'
import requests
import time
import sys
from concurrent.futures import ThreadPoolExecutor

url = sys.argv[1] if len(sys.argv) > 1 else "https://api.github.com/graphql"
iterations = int(sys.argv[2]) if len(sys.argv) > 2 else 100
concurrent = int(sys.argv[3]) if len(sys.argv) > 3 else 10

query = "{ __typename }"
headers = {"Content-Type": "application/json"}

def make_request(i):
    start = time.time()
    try:
        resp = requests.post(url, json={"query": query}, headers=headers, timeout=30)
        elapsed = time.time() - start
        return elapsed, resp.status_code
    except Exception as e:
        return time.time() - start, 0

# Sequential test
print(f"\nPython Sequential ({iterations} requests)...")
start = time.time()
success = 0
for i in range(iterations):
    elapsed, status = make_request(i)
    if status == 200:
        success += 1
sequential_time = time.time() - start
sequential_rps = iterations / sequential_time

print(f"Sequential: {sequential_rps:.2f} req/sec, {success}/{iterations} success")

# Concurrent test
print(f"\nPython Concurrent ({concurrent} workers, {iterations} requests)...")
start = time.time()
success = 0
with ThreadPoolExecutor(max_workers=concurrent) as executor:
    results = list(executor.map(make_request, range(iterations)))
    for elapsed, status in results:
        if status == 200:
            success += 1
concurrent_time = time.time() - start
concurrent_rps = iterations / concurrent_time

print(f"Concurrent: {concurrent_rps:.2f} req/sec, {success}/{iterations} success")

# Output for parsing
print(f"\nPYTHON_SEQUENTIAL_RPS:{sequential_rps:.2f}")
print(f"PYTHON_CONCURRENT_RPS:{concurrent_rps:.2f}")
EOF

    python3 python_benchmark.py "$GRAPHQL_URL" "$ITERATIONS" "$CONCURRENT" | tee python_benchmarks.txt
    
    echo ""
    echo -e "${GREEN}=== Python Performance Summary ===${NC}"
    grep "PYTHON_" python_benchmarks.txt
    
    echo ""
    echo -e "${GREEN}=== Comparison Summary ===${NC}"
    
    # Extract Go RPS (from concurrent fuzzing benchmark)
    GO_RPS=$(grep "BenchmarkConcurrentFuzzing" go_benchmarks.txt | tail -1 | awk '{print $5}')
    PYTHON_RPS=$(grep "PYTHON_CONCURRENT_RPS" python_benchmarks.txt | cut -d: -f2)
    
    if [ -n "$GO_RPS" ] && [ -n "$PYTHON_RPS" ]; then
        SPEEDUP=$(echo "scale=2; $GO_RPS / $PYTHON_RPS" | bc 2>/dev/null || echo "N/A")
        echo -e "Go concurrent performance: ${GO_RPS} ops/sec"
        echo -e "Python concurrent performance: ${PYTHON_RPS} req/sec"
        echo -e "${GREEN}Go is approximately ${SPEEDUP}x faster${NC}"
    fi
else
    echo ""
    echo -e "${YELLOW}Python GraphQLmap not found. Skipping Python comparison.${NC}"
    echo "To compare, clone the Python version: git clone https://github.com/swisskyrepo/GraphQLmap ../GraphQLmap"
fi

echo ""
echo -e "${GREEN}=== Memory Usage Comparison ===${NC}"

# Go memory usage
echo "Go binary size:"
ls -lh graphqlmap-go | awk '{print $5}'

if [ "$PYTHON_MAP_EXISTS" = true ]; then
    echo "Python dependencies:"
    du -sh ../GraphQLmap 2>/dev/null || echo "N/A"
fi

echo ""
echo -e "${GREEN}=== Test Results ===${NC}"
go test -v -race ./... 2>&1 | grep -E "(PASS|FAIL|ok|---)" | tail -20

echo ""
echo -e "${GREEN}Benchmarks saved to:${NC}"
echo "  - go_benchmarks.txt"
if [ "$PYTHON_MAP_EXISTS" = true ]; then
    echo "  - python_benchmarks.txt"
fi

echo ""
echo -e "${GREEN}Done!${NC}"
