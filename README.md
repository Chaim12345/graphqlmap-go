# GraphQLmap Go

A Go port of [GraphQLmap](https://github.com/swisskyrepo/GraphQLmap) - an interactive scripting engine for pentesting GraphQL endpoints.

## Features

- ✅ Interactive REPL with command history
- ✅ Schema dumping via introspection
- ✅ Custom HTTP headers, methods, and proxy support
- ✅ Fuzzing with special tokens:
  - `GRAPHQL_INCREMENT` - Incremental fuzzing
  - `GRAPHQL_CHARSET` - Character set fuzzing
  - `BLIND_PLACEHOLDER` - Blind injection testing
- ✅ Built-in injection testers:
  - NoSQLi
  - PostgreSQL
  - MySQL
  - MSSQL
- ✅ Query batching support
- ✅ Concurrent fuzzing with configurable workers
- ✅ Single static binary (no Python dependencies)

## Installation

```bash
git clone https://github.com/Chaim12345/graphqlmap-go.git
cd graphqlmap-go
go build -o graphqlmap-go
```

## Usage

### Basic

```bash
./graphqlmap-go -url https://example.com/graphql
```

### With Proxy (Burp Suite)

```bash
./graphqlmap-go -url https://example.com/graphql -proxy http://127.0.0.1:8080
```

### With Custom Headers

```bash
./graphqlmap-go -url https://example.com/graphql -H "Authorization: Bearer token, X-Custom-Header: value"
```

### All Options

```bash
./graphqlmap-go -help
  -H string
        Custom headers (e.g., 'Authorization: Bearer token')
  -X string
        HTTP method (POST or GET) (default "POST")
  -content-type string
        Content-Type header (default "application/json")
  -e string
        Encoding (json or form) (default "json")
  -proxy string
        HTTP proxy (e.g., http://127.0.0.1:8080)
  -proxy-auth string
        Proxy authentication credentials
  -url string
        GraphQL endpoint URL
```

## Interactive Commands

Once running, you can use these commands:

- `dump` - Dump the GraphQL schema using introspection
- `help` - Show help message
- `exit` - Exit the tool

## Special Tokens

Use these tokens in your queries for automated fuzzing:

### GRAPHQL_INCREMENT

Tests incremental values (e.g., `test1`, `test2`, `test3`...):

```graphql
{ user(id: "GRAPHQL_INCREMENT") { name email } }
```

### GRAPHQL_CHARSET

Tests character-by-character fuzzing:

```graphql
query { search(query: "GRAPHQL_CHARSET") { results } }
```

### BLIND_PLACEHOLDER

Placeholder for blind injection testing:

```graphql
{ user(id: "BLIND_PLACEHOLDER") { name } }
```

## Examples

### Dump Schema

```
graphqlmap> dump
Dumping schema...
Data:
{
  "__schema": {
    "queryType": { "name": "Query" },
    "types": [...]
  }
}
```

### Simple Query

```
graphqlmap> { __typename }
Data:
{
  "__typename": "Query"
}
```

### Fuzzing Example

```graphql
{ user(id: "GRAPHQL_INCREMENT") { name } }
```

This will test `user(id: "0")`, `user(id: "1")`, `user(id: "2")`, etc. concurrently and highlight interesting responses.

## Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o graphqlmap-go-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o graphqlmap-go.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o graphqlmap-go-mac

# ARM (for Raspberry Pi, etc.)
GOOS=linux GOARCH=arm64 go build -o graphqlmap-go-arm
```

## Differences from Python Version

- Uses `readline` library instead of `prompt_toolkit` for the REPL
- Concurrent fuzzing with goroutines (Python version is sequential)
- Single binary distribution (no pip install required)
- Built-in injection testers as separate module
- No external dependencies beyond Go standard library + readline

## License

MIT - Same as original GraphQLmap

## Disclaimer

For educational and authorized security testing only. Do not use for illegal testing.

## Credits

Original concept and Python implementation by [@swisskyrepo](https://github.com/swisskyrepo/GraphQLmap)

This is a Go port with additional features for concurrent fuzzing and injection testing.
