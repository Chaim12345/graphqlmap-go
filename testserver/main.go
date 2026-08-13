package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{}            `json:"data,omitempty"`
	Errors []map[string]interface `json:"errors,omitempty"`
}

type Schema struct {
	Types       []Type `json:"types"`
	QueryType   Type   `json:"queryType"`
	MutationType Type   `json:"mutationType"`
}

type Type struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Fields []Field `json:"fields,omitempty"`
}

type Field struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
}

func main() {
	port := ":8080"
	if p := getEnv("PORT", ""); p != "" {
		port = ":" + p
	}

	http.HandleFunc("/graphql", handleGraphQL)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Demo GraphQL server starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GraphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid JSON")
		return
	}

	response := executeQuery(req.Query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func executeQuery(query string) GraphQLResponse {
	query = strings.TrimSpace(query)

	// Handle introspection
	if strings.Contains(query, "__schema") {
		return GraphQLResponse{
			Data: map[string]interface{}{
				"__schema": map[string]interface{}{
					"queryType": map[string]string{"name": "Query"},
					"mutationType": map[string]string{"name": "Mutation"},
					"types": []Type{
						{Name: "Query", Kind: "OBJECT"},
						{Name: "Mutation", Kind: "OBJECT"},
						{Name: "User", Kind: "OBJECT"},
						{Name: "String", Kind: "SCALAR"},
						{Name: "Int", Kind: "SCALAR"},
					},
				},
			},
		}
	}

	// Handle __typename
	if strings.Contains(query, "__typename") {
		return GraphQLResponse{
			Data: map[string]string{"__typename": "Query"},
		}
	}

	// Handle user queries
	if strings.Contains(query, "user") {
		// Simulate different responses for fuzzing
		if strings.Contains(query, "test1") || strings.Contains(query, "test3") {
			return GraphQLResponse{
				Data: map[string]interface{}{
					"user": map[string]string{"id": "123", "name": "Interesting User"},
				},
			}
		}
		return GraphQLResponse{
			Data: map[string]interface{}{
				"user": map[string]string{"id": "456", "name": "Test User"},
			},
		}
	}

	// Handle search queries
	if strings.Contains(query, "search") {
		return GraphQLResponse{
			Data: map[string]interface{}{
				"search": []map[string]string{
					{"id": "1", "name": "Result 1"},
					{"id": "2", "name": "Result 2"},
				},
			},
		}
	}

	// Handle injection tests
	if strings.Contains(query, "DROP TABLE") || strings.Contains(query, "OR '1'='1") {
		return GraphQLResponse{
			Errors: []map[string]interface{}{
				{"message": "Syntax Error: Unexpected Name", "locations": []map[string]int{{"line": 1, "column": 2}}},
			},
		}
	}

	// Handle blind injection with delay
	if strings.Contains(query, "sleep") || strings.Contains(query, "WAITFOR") {
		time.Sleep(6 * time.Second)
		return GraphQLResponse{
			Data: map[string]string{"result": "delayed"},
		}
	}

	// Default response
	return GraphQLResponse{
		Data: map[string]string{"result": "ok"},
	}
}

func sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(GraphQLResponse{
		Errors: []map[string]interface{}{
			{"message": message},
		},
	})
}

func getEnv(key, defaultValue string) string {
	// Simple env getter (in real code, use os.Getenv)
	_ = key
	return defaultValue
}
