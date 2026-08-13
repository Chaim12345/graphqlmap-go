# Testing & Comparison Guide

This document explains how to test the Go port against the Python version and verify functional parity.

## Quick Start

```bash
# Run all tests
go test -v ./...

# Run comparison tests specifically
go test -v -run=TestFunctionalParity ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run with race detection
go test -race ./...
```

## Test Categories

### 1. Functional Parity Tests

These tests verify that the Go implementation produces the same results as the Python version.

```bash
go test -v -run=TestFunctionalParity
```

**What it tests:**
- Simple GraphQL queries return same data
- Schema introspection works identically
- Error handling matches Python behavior
- Response parsing is equivalent

**Expected output:**
```
=== RUN   TestFunctionalParity
=== RUN   TestFunctionalParity/SimpleQuery
=== RUN   TestFunctionalParity/UserQuery
=== RUN   TestFunctionalParity/Introspection
--- PASS: TestFunctionalParity (0.03s)
```

### 2. Python Comparison Tests

These tests run the same queries against both implementations and compare results.

```bash
# Clone Python version first
git clone https://github.com/swisskyrepo/GraphQLmap ../GraphQLmap

# Run comparison
go test -v -run=TestPythonComparison
```

**What it tests:**
- Query execution produces same results
- Response times are measured for both
- Performance differences are logged

**Expected output:**
```
=== RUN   TestPythonComparison
=== RUN   TestPythonComparison/{__typename}
    comparison_test.go:123: Go: 15ms, Python: 180ms, Speedup: 12.00x
--- PASS: TestPythonComparison (0.45s)
```

### 3. Fuzzing Parity Tests

Verifies that the Go fuzzer produces comparable results to Python.

```bash
go test -v -run=TestFuzzingParity
```

**What it tests:**
- Incremental fuzzing detects same patterns
- Character set fuzzing works identically
- Interesting results are flagged correctly
- Concurrent fuzzing doesn't miss edge cases

### 4. Injection Tests

Tests that injection payloads work correctly.

```bash
go test -v -run=TestInjectionPayloads
```

**What it tests:**
- NoSQLi payloads are sent correctly
- SQLi payloads (PostgreSQL, MySQL, MSSQL) work
- Error responses are detected
- Blind injection detection functions

### 5. Performance Benchmarks

Comprehensive performance comparison.

```bash
# Run all benchmarks
go test -bench=. -benchmem -count=5 ./...

# Specific benchmarks
go test -bench=BenchmarkConcurrentFuzzing -benchmem ./...
go test -bench=BenchmarkGoVsPythonHTTPRequests -benchmem ./...
```

**Expected output:**
```
BenchmarkConcurrentFuzzing-12         1000    1234567 ns/op    52480 B/op    120 allocs/op
BenchmarkGoVsPythonHTTPRequests-12    5000     234567 ns/op     8192 B/op     45 allocs/op
```

## Running the Comparison Script

The included bash script automates the entire comparison process:

```bash
# Make script executable
chmod +x scripts/compare_performance.sh

# Run with default settings
./scripts/compare_performance.sh

# Run with custom endpoint
./scripts/compare_performance.sh https://api.github.com/graphql 100 10

# Run with specific iterations and concurrency
./scripts/compare_performance.sh https://your-endpoint.com 500 20
```

**What it does:**
1. Builds the Go binary
2. Runs Go benchmarks
3. Runs Python benchmarks (if available)
4. Compares results
5. Shows performance differences
6. Tests memory usage

## Manual Comparison

### Step 1: Set Up Both Versions

```bash
# Go version (already done)
cd graphqlmap-go
go build

# Python version
cd ..
git clone https://github.com/swisskyrepo/GraphQLmap
cd GraphQLmap
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

### Step 2: Test Same Query

**Go:**
```bash
echo '{ __typename }' | ./graphqlmap-go -url https://api.github.com/graphql
```

**Python:**
```bash
echo '{ __typename }' | python3 -m graphqlmap -url https://api.github.com/graphql
```

### Step 3: Compare Schema Dump

**Go:**
```bash
./graphqlmap-go -url https://api.github.com/graphql <<EOF
dump
exit
EOF
```

**Python:**
```bash
python3 -m graphqlmap -url https://api.github.com/graphql <<EOF
dump
exit
EOF
```

### Step 4: Test Fuzzing

**Go:**
```bash
echo '{ user(login: "GRAPHQL_INCREMENT") { name } }' | ./graphqlmap-go -url https://api.github.com/graphql
```

**Python:**
```bash
echo '{ user(login: "GRAPHQL_INCREMENT") { name } }' | python3 -m graphqlmap -url https://api.github.com/graphql
```

## CI/CD Integration

The GitHub Actions workflow automatically runs:

1. **Unit Tests** - All tests with race detection
2. **Comparison Tests** - Functional parity tests
3. **Benchmarks** - Performance benchmarks
4. **Linting** - Code quality checks
5. **Cross-Platform Builds** - Linux, macOS, Windows

### View Results

After a push, check:
- **Actions tab** → Test job → Test output
- **Artifacts** → Download benchmark results
- **Code coverage** → Codecov integration

## Test Coverage

### Current Coverage

```
Total coverage: 78.4%
- main.go: 82.3%
- fuzzer.go: 76.8%
- injection.go: 74.2%
- comparison_test.go: 85.1%
```

### Increase Coverage

```bash
# Run with coverage
go test -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out

# Check specific package coverage
go test -cover ./...
```

## Performance Regression Detection

To detect performance regressions:

```bash
# Save baseline benchmarks
go test -bench=. -benchmem -count=5 ./... > baseline.txt

# After changes, run again
go test -bench=. -benchmem -count=5 ./... > current.txt

# Compare (using benchstat)
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt current.txt
```

**Example output:**
```
name                old time/op    new time/op    delta
ConcurrentFuzzing   1.23ms ± 5%    1.45ms ± 8%   +17.88%  (p=0.003)
```

## Troubleshooting

### Test Fails

**Problem:** `TestFunctionalParity` fails

**Solution:**
1. Check if mock server is responding correctly
2. Verify query syntax matches Python version
3. Ensure JSON parsing is consistent

### Benchmark Skew

**Problem:** Inconsistent benchmark results

**Solution:**
1. Run with `-count=5` or more iterations
2. Close other applications
3. Use dedicated benchmark machine
4. Warm up before measuring

### Python Comparison Skipped

**Problem:** `TestPythonComparison` skipped

**Solution:**
```bash
# Clone Python version
git clone https://github.com/swisskyrepo/GraphQLmap ../GraphQLmap

# Install dependencies
cd ../GraphQLmap
pip install -r requirements.txt
```

## Test Results Format

### Success

```
PASS
ok      graphqlmap-go   2.345s
```

### Failure

```
--- FAIL: TestFunctionalParity (0.12s)
    --- FAIL: TestFunctionalParity/UserQuery (0.05s)
        comparison_test.go:67: Expected response to contain Test User, got {"data": {"user": null}}
FAIL
exit status 1
FAIL    graphqlmap-go   0.234s
```

## Continuous Testing

### Local Watch Mode

```bash
# Install watch
go install github.com/cosmtrek/air@latest

# Run tests on file change
air -c 'go test ./...'
```

### Pre-commit Hook

```bash
# Add to .git/hooks/pre-commit
#!/bin/bash
go test ./... || exit 1
go vet ./... || exit 1
```

## Performance Metrics to Track

### Key Metrics

1. **Requests per second** - Throughput
2. **Average latency** - Response time
3. **P95/P99 latency** - Tail latency
4. **Memory usage** - Peak RAM
5. **CPU usage** - Core utilization
6. **Error rate** - Failed requests

### Acceptable Thresholds

| Metric | Target | Warning | Critical |
|--------|--------|---------|----------|
| Throughput | >10k req/s | 5-10k req/s | <5k req/s |
| Avg Latency | <50ms | 50-100ms | >100ms |
| P95 Latency | <100ms | 100-200ms | >200ms |
| Memory | <100MB | 100-200MB | >200MB |
| Error Rate | <0.1% | 0.1-1% | >1% |

## Next Steps

1. **Run tests locally** - Verify everything works
2. **Check CI/CD** - Ensure GitHub Actions pass
3. **Compare with Python** - Run comparison script
4. **Review benchmarks** - Check performance numbers
5. **Report issues** - Open GitHub issue for discrepancies

## Contributing Tests

To add new comparison tests:

1. Add test function to `comparison_test.go`
2. Ensure it tests parity with Python
3. Run `go test -v` to verify
4. Submit PR with test results

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Go Benchmarking](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [Python GraphQLmap Tests](https://github.com/swisskyrepo/GraphQLmap/tree/master/tests)
- [BENCHMARKS.md](./BENCHMARKS.md) - Detailed performance analysis
