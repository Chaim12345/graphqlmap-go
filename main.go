package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

var (
	urlFlag       = flag.String("url", "", "GraphQL endpoint URL")
	methodFlag    = flag.String("X", "POST", "HTTP method (POST or GET)")
	proxyFlag     = flag.String("proxy", "", "HTTP proxy (e.g., http://127.0.0.1:8080)")
	headersFlag   = flag.String("H", "", "Custom headers (e.g., 'Authorization: Bearer token')")
	contentType   = flag.String("content-type", "application/json", "Content-Type header")
	encodingFlag  = flag.String("e", "json", "Encoding (json or form)")
	proxyAuthFlag = flag.String("proxy-auth", "", "Proxy authentication credentials")
)

type Config struct {
	URL         string
	Method      string
	Proxy       string
	Headers     map[string]string
	ContentType string
	Encoding    string
	ProxyAuth   string
}

type GraphQLResponse struct {
	Data       json.RawMessage        `json:"data"`
	Errors     []map[string]interface `json:"errors"`
	Extensions map[string]interface   `json:"extensions"`
}

func parseHeaders(headerStr string) map[string]string {
	headers := make(map[string]string)
	if headerStr == "" {
		return headers
	}
	parts := strings.Split(headerStr, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) == 2 {
			headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return headers
}

func loadConfig() *Config {
	flag.Parse()
	return &Config{
		URL:         *urlFlag,
		Method:      strings.ToUpper(*methodFlag),
		Proxy:       *proxyFlag,
		Headers:     parseHeaders(*headersFlag),
		ContentType: *contentType,
		Encoding:    *encodingFlag,
		ProxyAuth:   *proxyAuthFlag,
	}
}

func createHTTPClient(config *Config) *http.Client {
	transport := &http.Transport{}
	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &http.Client{Transport: transport}
}

func executeQuery(config *Config, client *http.Client, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	var req *http.Request
	var err error

	queryBody := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		queryBody["variables"] = variables
	}

	var bodyReader io.Reader
	var contentType string

	if config.Encoding == "json" {
		jsonData, err := json.Marshal(queryBody)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonData)
		contentType = "application/json"
	} else {
		formData := url.Values{}
		formData.Set("query", query)
		if variables != nil {
			varJSON, _ := json.Marshal(variables)
			formData.Set("variables", string(varJSON))
		}
		bodyReader = strings.NewReader(formData.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	if config.Method == "POST" {
		req, err = http.NewRequest("POST", config.URL, bodyReader)
	} else {
		req, err = http.NewRequest("GET", config.URL, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	if config.ProxyAuth != "" {
		req.Header.Set("Proxy-Authorization", "Basic "+config.ProxyAuth)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	return &gqlResp, nil
}

func printResponse(resp *GraphQLResponse) {
	if resp.Errors != nil && len(resp.Errors) > 0 {
		color.Red("Errors:")
		for _, e := range resp.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}
	if resp.Data != nil {
		color.Green("Data:")
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, resp.Data, "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
		} else {
			fmt.Println(string(resp.Data))
		}
	}
}

func dumpSchema(config *Config, client *http.Client) {
	introspectionQuery := `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type {
    ...TypeRef
  }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`
	color.Yellow("Dumping schema...")
	resp, err := executeQuery(config, client, introspectionQuery, nil)
	if err != nil {
		color.Red("Error: %v", err)
		return
	}
	printResponse(resp)
}

func runInteractive(config *Config) {
	client := createHTTPClient(config)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          color.GreenString("graphqlmap> "),
		HistoryFile:     "/tmp/graphqlmap_history.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		HistorySearchFilter: true,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nExiting...")
		os.Exit(0)
	}()

	color.Cyan("GraphQLmap Go - Interactive GraphQL Pentesting Tool")
	color.Cyan("Commands: dump, help, exit")
	color.Cyan("Special tokens: GRAPHQL_INCREMENT, GRAPHQL_CHARSET, BLIND_PLACEHOLDER")
	fmt.Println()

	for {
		rl.Refresh()
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				fmt.Println("\nExiting...")
				break
			}
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case "exit", "quit":
			fmt.Println("Exiting...")
			return
		case "help":
			printHelp()
			continue
		case "dump":
			dumpSchema(config, client)
			continue
		}

		resp, err := executeQuery(config, client, line, nil)
		if err != nil {
			color.Red("Error: %v", err)
			continue
		}
		printResponse(resp)
	}
}

func printHelp() {
	fmt.Println(`
Commands:
  dump              Dump the GraphQL schema using introspection
  help              Show this help message
  exit              Exit the tool

Special Tokens:
  GRAPHQL_INCREMENT   - Incremental fuzzing (e.g., test1, test2, test3...)
  GRAPHQL_CHARSET     - Character set fuzzing (e.g., a, b, c... or 0, 1, 2...)
  BLIND_PLACEHOLDER   - Placeholder for blind injection testing

Examples:
  { user(id: "GRAPHQL_INCREMENT") { name } }
  query { search(query: "GRAPHQL_CHARSET") { results } }
`)
}

func main() {
	config := loadConfig()

	if config.URL == "" {
		fmt.Println("Usage: graphqlmap-go -url <GraphQL_endpoint> [options]")
		fmt.Println("Run with -help for more options")
		os.Exit(1)
	}

	runInteractive(config)
}
