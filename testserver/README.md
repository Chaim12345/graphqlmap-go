# Demo GraphQL Server

A simple GraphQL server for testing both Go and Python implementations.

## Running Locally

```bash
go run testserver/main.go
```

Server will start on `http://localhost:8080/graphql`

## Running with Docker

```bash
docker build -t graphql-demo-server .
docker run -p 8080:8080 graphql-demo-server
```

## Endpoints

- `POST /graphql` - GraphQL endpoint
- `GET /health` - Health check

## Test Queries

```graphql
# Simple query
{ __typename }

# Introspection
query { __schema { types { name } } }

# User query
{ user(id: "123") { id name } }

# Search
{ search(query: "test") { id name } }
```

## Features

- Returns consistent responses for testing
- Simulates different response times for blind injection testing
- Handles introspection queries
- Returns errors for injection payloads
- Configurable port via PORT environment variable
