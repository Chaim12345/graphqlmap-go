# CI/CD Setup Guide

This document explains how the CI/CD pipeline works and how to run tests locally.

## Architecture

The CI pipeline uses a demo GraphQL server that both Go and Python implementations test against:

```
┌─────────────────┐
│  Demo Server    │
│  (port 8080)    │
│                 │
│  - Go version   │
│  - Python compat│
└────────┬────────┘
         │
         ├──────────────────┐
         │                  │
┌────────▼────────┐ ┌──────▼──────┐
│  Go Tests       │ │ Python Tests│
│                 │ │             │
│  - Unit tests   │ │ - requests  │
│  - Comparison   │ │ - CLI tool  │
│  - Benchmarks   │ │             │
└─────────────────┘ └─────────────┘
```

## CI Jobs

### 1. Test & Build Job

**Runs on:** `ubuntu-latest`

**Services:**
- Demo GraphQL server (Go-based, port 8080)

**Steps:**
1. Checkout code
2. Set up Go 1.23
3. Set up Python 3.11
4. Start demo server
5. Download Go dependencies
6. Build project
7. Run unit tests with race detection
8. Run comparison tests
9. Run integration tests
10. Upload coverage to Codecov

**Environment Variables:**
```yaml
TEST_GRAPHQL_URL: http://localhost:8080/graphql
```

### 2. Python Comparison Job

**Runs on:** `ubuntu-latest`

**Services:**
- Demo GraphQL server (same as above)

**Steps:**
1. Checkout code
2. Set up Go 1.23
3. Set up Python 3.11
4. Start demo server
5. Install Python dependencies (requests)
6. Clone Python GraphQLmap
7. Run Python comparison tests
8. Manual Python vs Go comparison

### 3. Lint Job

**Runs on:** `ubuntu-latest`

**Steps:**
1. Checkout code
2. Set up Go 1.23
3. Install golangci-lint
4. Run linter

### 4. Benchmark Job

**Runs on:** `ubuntu-latest`

**Services:**
- Demo GraphQL server

**Steps:**
1. Checkout code
2. Set up Go 1.23
3. Set up Python 3.11
4. Start demo server
5. Run benchmarks
6. Upload benchmark results as artifacts
7. Check for performance regressions

### 5. Cross-Platform Build Job

**Runs on:** `ubuntu-latest`

**Matrix:**
- linux/amd64, linux/arm64
- darwin/amd64
- windows/amd64, windows/arm64

**Steps:**
1. Checkout code
2. Set up Go 1.23
3. Build for each platform
4. Upload artifacts

## Running Tests Locally

### Option 1: Using Docker Compose

```bash
# Start demo server and run tests
docker-compose up test-runner

# Run Python tests
docker-compose up python-test

# Run everything
docker-compose up
```

### Option 2: Using Shell Script

```bash
# Make script executable
chmod +x scripts/run_local_tests.sh

# Run all tests
./scripts/run_local_tests.sh
```

### Option 3: Manual Setup

```bash
# 1. Start demo server
cd testserver
go mod init testserver
go build
./testserver &

# 2. Wait for server
sleep 2
curl http://localhost:8080/health

# 3. Set environment variable
export TEST_GRAPHQL_URL="http://localhost:8080/graphql"

# 4. Run tests
go test -v ./...

# 5. Run benchmarks
go test -bench=. -benchmem ./...

# 6. Kill demo server
kill %1
```

## Demo Server Endpoints

### Health Check

```bash
curl http://localhost:8080/health
# Response: {"status":"ok"}
```

### GraphQL Endpoint

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __typename }"}'
# Response: {"data":{"__typename":"Query"}}
```

## Test Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_GRAPHQL_URL` | GraphQL endpoint URL | Auto-created test server |
| `PORT` | Demo server port | 8080 |

## CI Configuration Details

### Service Container Health Check

The demo server uses a health check to ensure it's ready before tests run:

```yaml
options: >-
  --health-cmd "wget -q --spider http://localhost:8080/health || exit 1"
  --health-interval 10s
  --health-timeout 5s
  --health-retries 5
```

### Python Setup

Python is set up with caching for faster installs:

```yaml
- name: Set up Python
  uses: actions/setup-python@v5
  with:
    python-version: '3.11'
    cache: 'pip'
```

### Go Setup

Go uses module caching:

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.23'
    cache: true
```

## Troubleshooting

### Demo Server Won't Start

```bash
# Check if port is available
netstat -tlnp | grep 8080

# Try different port
export PORT=8081
./testserver
```

### Tests Fail in CI

1. Check service container logs in GitHub Actions
2. Verify health check passes
3. Check if TEST_GRAPHQL_URL is set correctly
4. Run tests locally with same environment

### Python Comparison Fails

```bash
# Install Python dependencies
pip install requests

# Verify Python version
python3 --version

# Test manually
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __typename }"}'
```

### Benchmarks Show Regression

1. Download benchmark artifacts from CI
2. Compare with previous runs using `benchstat`
3. Check if demo server had issues
4. Re-run benchmarks locally

## Performance Expectations

### Test Duration

| Job | Expected Time |
|-----|---------------|
| Test & Build | 2-3 minutes |
| Python Comparison | 2-3 minutes |
| Lint | 1 minute |
| Benchmarks | 3-5 minutes |
| Cross-Platform Build | 5-7 minutes |
| **Total** | **~15 minutes** |

### Resource Usage

- **Concurrent Jobs:** 4 (test, python, lint, benchmark)
- **Service Containers:** 1 per job (demo server)
- **Disk Usage:** ~500MB per job

## Adding New Tests

### Unit Tests

Add to existing test files:

```go
func TestMyNewFeature(t *testing.T) {
    testURL := getTestURL(t)
    // Test logic here
}
```

### Integration Tests

Use the demo server:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    testURL := getTestURL(t)
    // Integration test logic
}
```

### Benchmarks

```go
func BenchmarkMyFeature(b *testing.B) {
    testURL := getTestURL(&testing.T{})
    // Benchmark logic
}
```

## Monitoring CI Health

### GitHub Actions Tab

1. Go to repository → Actions
2. Check workflow runs
3. View individual job logs
4. Download artifacts

### Coverage Reports

1. Check Codecov integration
2. View coverage trends
3. Identify uncovered code

### Benchmark Trends

1. Download benchmark artifacts
2. Compare with `benchstat`
3. Track performance over time

## Best Practices

1. **Always use `getTestURL()`** - Automatically handles local vs CI environment
2. **Mark slow tests** - Use `t.Parallel()` and `testing.Short()`
3. **Clean up resources** - Use `t.Cleanup()` for servers/files
4. **Test idempotently** - Tests should be runnable multiple times
5. **Document new tests** - Add to this file

## Related Files

- `.github/workflows/ci.yml` - Main CI configuration
- `testserver/main.go` - Demo GraphQL server
- `testserver/server.py` - Python version of demo server
- `comparison_test.go` - Comparison tests
- `docker-compose.yml` - Local Docker setup
- `scripts/run_local_tests.sh` - Local test runner

## Support

For issues with CI:
1. Check this documentation
2. Review GitHub Actions logs
3. Run tests locally
4. Open GitHub issue with logs
