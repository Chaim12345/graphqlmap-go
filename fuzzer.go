package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	TokenIncrement = "GRAPHQL_INCREMENT"
	TokenCharset   = "GRAPHQL_CHARSET"
	TokenBlind     = "BLIND_PLACEHOLDER"
)

type Fuzzer struct {
	config      *Config
	client      *http.Client
	baseQuery   string
	results     []FuzzResult
	mu          sync.Mutex
	concurrency int
	timeout     time.Duration
}

type FuzzResult struct {
	Input          string
	ResponseLength int
	ResponseTime   time.Duration
	Error          error
	Interesting    bool
	Notes          string
}

func NewFuzzer(config *Config, baseQuery string) *Fuzzer {
	return &Fuzzer{
		config:      config,
		client:      createHTTPClient(config),
		baseQuery:   baseQuery,
		concurrency: 10,
		timeout:     10 * time.Second,
	}
}

func (f *Fuzzer) FuzzIncrement(prefix string, start, end int) []FuzzResult {
	color.Yellow("Fuzzing with GRAPHQL_INCREMENT: %s%d to %s%d", prefix, start, prefix, end)

	var wg sync.WaitGroup
	sem := make(chan struct{}, f.concurrency)

	for i := start; i <= end; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(val int) {
			defer wg.Done()
			defer func() { <-sem }()

			input := fmt.Sprintf("%s%d", prefix, val)
			query := strings.ReplaceAll(f.baseQuery, TokenIncrement, input)

			startTime := time.Now()
			resp, err := executeQuery(f.config, f.client, query, nil)
			duration := time.Since(startTime)

			result := FuzzResult{
				Input:        input,
				ResponseTime: duration,
				Error:        err,
			}

			if err != nil {
				result.Notes = fmt.Sprintf("Error: %v", err)
			} else if resp != nil {
				dataLen := len(resp.Data)
				result.ResponseLength = dataLen
				result.Interesting = f.isInteresting(dataLen, duration)
				if result.Interesting {
					result.Notes = fmt.Sprintf("Length: %d, Time: %v", dataLen, duration)
				}
			}

			f.mu.Lock()
			f.results = append(f.results, result)
			f.mu.Unlock()

			if result.Interesting {
				color.Green("[+] Interesting: %s - %s", input, result.Notes)
			}
		}(i)
	}

	wg.Wait()
	return f.results
}

func (f *Fuzzer) FuzzCharset(charset string) []FuzzResult {
	color.Yellow("Fuzzing with GRAPHQL_CHARSET: %s", charset)

	var wg sync.WaitGroup
	sem := make(chan struct{}, f.concurrency)

	for _, c := range charset {
		wg.Add(1)
		sem <- struct{}{}
		go func(ch rune) {
			defer wg.Done()
			defer func() { <-sem }()

			input := string(ch)
			query := strings.ReplaceAll(f.baseQuery, TokenCharset, input)

			startTime := time.Now()
			resp, err := executeQuery(f.config, f.client, query, nil)
			duration := time.Since(startTime)

			result := FuzzResult{
				Input:        input,
				ResponseTime: duration,
				Error:        err,
			}

			if err != nil {
				result.Notes = fmt.Sprintf("Error: %v", err)
			} else if resp != nil {
				dataLen := len(resp.Data)
				result.ResponseLength = dataLen
				result.Interesting = f.isInteresting(dataLen, duration)
				if result.Interesting {
					result.Notes = fmt.Sprintf("Length: %d, Time: %v", dataLen, duration)
				}
			}

			f.mu.Lock()
			f.results = append(f.results, result)
			f.mu.Unlock()

			if result.Interesting {
				color.Green("[+] Interesting: '%s' - %s", input, result.Notes)
			}
		}(ch)
	}

	wg.Wait()
	return f.results
}

func (f *Fuzzer) FuzzBlind(payloads []string) []FuzzResult {
	color.Yellow("Blind injection testing with %d payloads", len(payloads))

	baseLength := -1

	for _, payload := range payloads {
		query := strings.ReplaceAll(f.baseQuery, TokenBlind, payload)

		startTime := time.Now()
		resp, err := executeQuery(f.config, f.client, query, nil)
		duration := time.Since(startTime)

		result := FuzzResult{
			Input:        payload,
			ResponseTime: duration,
			Error:        err,
		}

		if err != nil {
			result.Notes = fmt.Sprintf("Error: %v", err)
		} else if resp != nil {
			dataLen := len(resp.Data)
			result.ResponseLength = dataLen

			if baseLength == -1 {
				baseLength = dataLen
			}

			lengthDiff := dataLen - baseLength
			if lengthDiff != 0 || duration > 5*time.Second {
				result.Interesting = true
				result.Notes = fmt.Sprintf("Length diff: %+d, Time: %v", lengthDiff, duration)
				color.Green("[+] Interesting: %s - %s", payload, result.Notes)
			}
		}

		f.mu.Lock()
		f.results = append(f.results, result)
		f.mu.Unlock()
	}

	return f.results
}

func (f *Fuzzer) isInteresting(length int, duration time.Duration) bool {
	if duration > 5*time.Second {
		return true
	}
	if len(f.results) == 0 {
		return false
	}

	avgLen := 0
	for _, r := range f.results {
		avgLen += r.ResponseLength
	}
	avgLen /= len(f.results)

	diff := length - avgLen
	return diff > 100 || diff < -100
}

func (f *Fuzzer) PrintResults() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	color.Cyan("Fuzzing Results:")
	fmt.Println(strings.Repeat("=", 80))

	interestingCount := 0
	for _, r := range f.results {
		if r.Interesting {
			interestingCount++
			fmt.Printf("[+] %s | Length: %d | Time: %v | %s\n",
				r.Input, r.ResponseLength, r.ResponseTime, r.Notes)
		}
	}

	fmt.Printf("\nTotal: %d results, %d interesting\n", len(f.results), interestingCount)
}
