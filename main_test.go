package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single header",
			input:    "Authorization: Bearer token",
			expected: map[string]string{"Authorization": "Bearer token"},
		},
		{
			name:     "multiple headers",
			input:    "Authorization: Bearer token, X-Custom: value",
			expected: map[string]string{"Authorization": "Bearer token", "X-Custom": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeaders(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d headers, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Header %s: expected %s, got %s", k, v, result[k])
				}
			}
		})
	}
}

func TestExecuteQuery(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"__typename": "Query"}}`))
	}))
	defer testServer.Close()

	config := &Config{
		URL:         testServer.URL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	client := createHTTPClient(config)
	resp, err := executeQuery(config, client, "{ __typename }", nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(resp.Data) == 0 {
		t.Error("Expected data in response")
	}
}

func TestCreateHTTPClient(t *testing.T) {
	config := &Config{
		Proxy: "http://127.0.0.1:8080",
	}

	client := createHTTPClient(config)
	if client == nil {
		t.Error("Expected client, got nil")
	}

	configNoProxy := &Config{}
	clientNoProxy := createHTTPClient(configNoProxy)
	if clientNoProxy == nil {
		t.Error("Expected client without proxy, got nil")
	}
}

func TestFuzzer(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"user": {"id": "test"}}}`))
	}))
	defer testServer.Close()

	config := &Config{
		URL:         testServer.URL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	fuzzer := NewFuzzer(config, `{ user(id: "GRAPHQL_INCREMENT") { id } }`)
	results := fuzzer.FuzzIncrement("test", 1, 3)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Error != nil {
			t.Errorf("Expected no error for result, got %v", r.Error)
		}
	}
}

func TestInjectionTester(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"user": {"id": "test"}}}`))
	}))
	defer testServer.Close()

	config := &Config{
		URL:         testServer.URL,
		Method:      "POST",
		ContentType: "application/json",
		Encoding:    "json",
	}

	tester := NewInjectionTester(config)

	tester.TestNoSQLi(`{ user(id: "BLIND_PLACEHOLDER") { id } }`)
	tester.TestPostgreSQL(`{ user(id: "BLIND_PLACEHOLDER") { id } }`)
	tester.TestMySQL(`{ user(id: "BLIND_PLACEHOLDER") { id } }`)
	tester.TestMSSQL(`{ user(id: "BLIND_PLACEHOLDER") { id } }`)
}
