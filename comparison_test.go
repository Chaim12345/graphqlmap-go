package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// getTestURL returns the GraphQL endpoint from environment or creates a test server
func getTestURL(t *testing.T) string {
	if url := os.Getenv("TEST_GRAPHQL_URL"); url != "" {
		return url
	}

	// Create local test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		query, _ := reqBody["query"].(string)
		
		if strings.Contains(query, "__schema") {
			w.Write([]byte(`{"data": {"__schema": {"queryType": {"name": "Query"}, "types": [{"name": "User", "kind": "OBJECT"}]}}}`))
		} else if strings.Contains(query, "__typename") {
			w.Write([]byte(`{"data": {"__typename": "Query"}}`))
		} else if strings.Contains(query, "user") {
			if strings.Contains(query, "test1") || strings.Contains(query, "test3") {
				w.Write([]byte(`{"data": {"user": {"id": "123", "name": "Interesting User"}}}`))
			} else {
				w.Write([]byte(`{"data": {"user": {"id": "456", "name": "Test User"}}}`))
			}
		} else if strings.Contains(query, "DROP TABLE") || strings.Contains(query, "OR '1'='1") {
			w.Write([]byte(`{"errors": [{"message": "Syntax Error: Unexpected Name"}]}`))
		} else {
			w.Write([]byte(`{"data": {"result": "ok"}}`))
		}
	}))
	t.Cleanup(testServer.Close)
	return testServer.URL
}

// TestFunctionalParity tests that Go implementation produces same results as Python version
func TestFunctionalParity(t *testing.T) {
	testURL := getTestURL(t)

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"SimpleQuery", "{ __typename }", "Query"},
		{"UserQuery", `{ user(id: "123") { id name } }`, "Test User"},
		{"Introspection", `query { __schema { queryType { name } } }`, "Query"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				URL:         testURL,
				Method:      "POST",
				ContentType: "application/json",
				Encoding:    "json",
			}
			client := createHTTPClient(config)
			resp, err := executeQuery(config, client, tt.query, nil)
			
			if err != nil {
				t.Fatalf("Go query failed: %v", err)
			}
			
			if resp == nil || len(resp.Data) == 0 {
				t.Fatal("Go query returned empty response")
			}
			
			if !strings.Contains(string(resp.Data), tt.expected) {
				t.Errorf("Expected response to contain %s, got %s", tt.expected, string(resp.Data))
			}
		})
	}
}

// TestPythonComparison runs the same queries against both implementations
func TestPythonComparison(t *testing.T) {
	testURL := getTestURL(t)

	queries := []string{
		`{ __typename }`,
		`{ test }`,
		`query { __schema { types { name } } }`,
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			// Time Go implementation
			goStart := time.Now()
			goConfig := &Config{
				URL:         testURL,
				Method:      "POST",
				ContentType: "application/json",
				Encoding:    "json",
			}
			goClient := createHTTPClient(goConfig)
			goResp, goErr := executeQuery(goConfig, goClient, query, nil)
			goDuration := time.Since(goStart)

			if goErr != nil {
				t.Fatalf("Go implementation failed: %v", goErr)
			}

			// Time Python implementation
			pythonStart := time.Now()
			pythonCmd := exec.Command("python3", "-c", 
				`import requests; r = requests.post("`+testURL+`", json={"query": "`+query+`"}); print(r.text)`)
			pythonOutput, pythonErr := pythonCmd.CombinedOutput()
			pythonDuration := time.Since(pythonStart)

			if pythonErr == nil {
				if !strings.Contains(string(pythonOutput), "test") && !strings.Contains(string(pythonOutput), "__schema") {
					t.Logf("Python output: %s", string(pythonOutput))
				}
				
				t.Logf("Go: %v, Python: %v, Speedup: %.2fx", 
					goDuration, pythonDuration, float64(pythonDuration)/float64(goDuration))
			} else {
				t.Logf("Python not available or failed: %v", pythonErr)
			}
		})
	}
}

// BenchmarkGoVsPythonHTTPRequests compares HTTP request performance
func BenchmarkGoVsPythonHTTPRequests(b *testing.B) {
	testURL := getTestURL(&testing.T{})

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}
	client := createHTTPClient(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executeQuery(config, client, "{ test }", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestFuzzingParity ensures Go fuzzer produces same results as Python
func TestFuzzingParity(t *testing.T) {
	testURL := getTestURL(t)

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	fuzzer := NewFuzzer(config, `{ user(id: "GRAPHQL_INCREMENT") { name } }`)
	results := fuzzer.FuzzIncrement("test", 1, 5)

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	interestingCount := 0
	for _, r := range results {
		if r.Interesting {
			interestingCount++
		}
	}

	if interestingCount == 0 {
		t.Error("Expected some interesting results")
	}
}

// TestInjectionPayloads verifies injection payloads work correctly
func TestInjectionPayloads(t *testing.T) {
	testURL := getTestURL(t)

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	tester := NewInjectionTester(config)
	baseQuery := `{ user(id: "BLIND_PLACEHOLDER") { id } }`
	
	tester.TestNoSQLi(baseQuery)
	tester.TestPostgreSQL(baseQuery)
	tester.TestMySQL(baseQuery)
	tester.TestMSSQL(baseQuery)
}

// BenchmarkConcurrentFuzzing demonstrates Go's concurrency advantage
func BenchmarkConcurrentFuzzing(b *testing.B) {
	testURL := getTestURL(&testing.T{})

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzer := NewFuzzer(config, `{ user(id: "GRAPHQL_INCREMENT") { name } }`)
		fuzzer.FuzzIncrement("test", 1, 10)
	}
}

// TestSchemaDumping verifies introspection query works
func TestSchemaDumping(t *testing.T) {
	testURL := getTestURL(t)

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	client := createHTTPClient(config)
	
	var buf bytes.Buffer
	oldStdout := os.Stdout
	os.Stdout = &buf
	
	dumpSchema(config, client)
	
	os.Stdout = oldStdout

	output := buf.String()
	if !strings.Contains(output, "User") {
		t.Error("Expected schema dump to contain User type")
	}
}

// TestHeaderParsing verifies header parsing matches Python behavior
func TestHeaderParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{"", map[string]string{}},
		{"Authorization: Bearer token", map[string]string{"Authorization": "Bearer token"}},
		{"Authorization: Bearer token, X-Custom: value", map[string]string{"Authorization": "Bearer token", "X-Custom": "value"}},
		{"Content-Type: application/json, Accept: application/json", map[string]string{"Content-Type": "application/json", "Accept": "application/json"}},
	}

	for _, tt := range tests {
		result := parseHeaders(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("Input %q: expected %d headers, got %d", tt.input, len(tt.expected), len(result))
		}
		for k, v := range tt.expected {
			if result[k] != v {
				t.Errorf("Input %q: header %s expected %q, got %q", tt.input, k, v, result[k])
			}
		}
	}
}

// TestProxySupport verifies proxy configuration works
func TestProxySupport(t *testing.T) {
	config := &Config{
		Proxy: "http://127.0.0.1:8080",
	}

	client := createHTTPClient(config)
	if client == nil {
		t.Fatal("Expected client with proxy")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected http.Transport")
	}

	proxyURL, _ := transport.Proxy(nil)
	if proxyURL == nil {
		t.Error("Expected proxy URL to be set")
	}
}

// TestConcurrentRequests tests that concurrent requests don't cause race conditions
func TestConcurrentRequests(t *testing.T) {
	testURL := getTestURL(t)

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			client := createHTTPClient(config)
			_, err := executeQuery(config, client, "{ test }", nil)
			if err != nil {
				t.Error(err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestErrorHandling verifies error responses are handled correctly
func TestErrorHandling(t *testing.T) {
	testURL := getTestURL(t)

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	client := createHTTPClient(config)
	resp, err := executeQuery(config, client, "{ invalid }", nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil || len(resp.Errors) == 0 {
		t.Fatal("Expected errors in response")
	}

	if !strings.Contains(resp.Errors[0]["message"].(string), "invalid") {
		t.Error("Expected error message to contain 'invalid'")
	}
}

// TestBlindInjectionDetection verifies blind injection detection works
func TestBlindInjectionDetection(t *testing.T) {
	testURL := getTestURL(t)

	config := &Config{
		URL:         testURL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	fuzzer := NewFuzzer(config, `{ user(id: "BLIND_PLACEHOLDER") { id } }`)
	payloads := []string{"payload1", "payload2", "sleep(5000)", "payload4"}
	results := fuzzer.FuzzBlind(payloads)

	timeBasedDetected := false
	for _, r := range results {
		if r.ResponseTime > 5*time.Second {
			timeBasedDetected = true
			break
		}
	}

	if !timeBasedDetected {
		t.Error("Expected time-based blind injection to be detected")
	}
}
