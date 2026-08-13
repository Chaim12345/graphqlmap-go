#!/bin/bash

# Run all tests locally with demo server

set -e

echo "=== Starting Demo Server ==="

# Start demo server in background
cd testserver
go mod init testserver 2>/dev/null || true
go build -o testserver main.go
./testserver &
DEMO_PID=$!
cd ..

# Wait for server to start
sleep 2

# Check if server is running
if ! curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "Failed to start demo server"
    kill $DEMO_PID 2>/dev/null
    exit 1
fi

echo "Demo server running on http://localhost:8080/graphql"
echo ""

echo "=== Running Go Tests ==="
export TEST_GRAPHQL_URL="http://localhost:8080/graphql"
go test -v ./...

echo ""
echo "=== Running Python Comparison ==="
python3 -c "
import requests
url = 'http://localhost:8080/graphql'

# Test simple query
resp = requests.post(url, json={'query': '{ __typename }'})
print('Python simple query:', resp.json())

# Test introspection
resp = requests.post(url, json={'query': 'query { __schema { types { name } } }'})
print('Python introspection:', 'types' in str(resp.json()))
"

echo ""
echo "=== Running Benchmarks ==="
go test -bench=. -benchmem -count=3 ./...

echo ""
echo "=== Cleaning Up ==="
kill $DEMO_PID 2>/dev/null || true
echo "Done!"
