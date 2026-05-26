package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// ============================================================
// PATTERN 8: HTTP Client Optimization
// ============================================================
// Problem: Default http.Client settings are not production-ready.
// Common mistakes:
// - No timeout (hangs forever on slow upstream)
// - Creating new client per request (no connection reuse)
// - Not reading/closing response body (leaks connections)
// - No context cancellation support
//
// This pattern demonstrates:
// 1. Proper timeout configuration
// 2. Transport tuning for connection reuse
// 3. Response body handling
// 4. Context-based cancellation
// ============================================================

// --- Bad Client (default settings) ---

// BadClient uses http.DefaultClient — no timeouts, default transport.
func BadClient() *http.Client {
	return http.DefaultClient
}

// --- Good Client (production-ready) ---

// GoodClient returns a properly configured HTTP client.
func GoodClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}
}

// --- Request Patterns ---

// BadRequest doesn't close body, no timeout, no context.
func BadRequest(client *http.Client, url string) (int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	// ❌ BAD: body not read/closed — connection can't be reused!
	return resp.StatusCode, nil
}

// GoodRequest properly handles response body and uses context.
func GoodRequest(ctx context.Context, client *http.Client, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// ✅ GOOD: drain body to allow connection reuse
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

// --- Concurrent Request Pattern ---

// FetchAll makes concurrent requests with a shared client.
func FetchAll(ctx context.Context, client *http.Client, urls []string) []int {
	results := make([]int, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			status, _ := GoodRequest(ctx, client, u)
			results[idx] = status
		}(i, url)
	}

	wg.Wait()
	return results
}

// --- Demonstration ---

func main() {
	fmt.Println("=== HTTP Client Optimization ===")
	fmt.Println()

	// Start test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond) // Simulate processing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	url := server.URL

	// 1. Bad client — connection leak demonstration
	fmt.Println("--- Bad Client (no body close) ---")
	badClient := BadClient()
	start := time.Now()
	for i := 0; i < 100; i++ {
		BadRequest(badClient, url)
	}
	fmt.Printf("100 requests (bad): %v\n", time.Since(start))
	fmt.Println("⚠️  Connections leaked — can't be reused")
	fmt.Println()

	// 2. Good client — connection reuse
	fmt.Println("--- Good Client (proper body handling) ---")
	goodClient := GoodClient()
	ctx := context.Background()
	start = time.Now()
	for i := 0; i < 100; i++ {
		GoodRequest(ctx, goodClient, url)
	}
	fmt.Printf("100 requests (good): %v\n", time.Since(start))
	fmt.Println("✅ Connections reused via keep-alive")
	fmt.Println()

	// 3. Concurrent requests
	fmt.Println("--- Concurrent Requests (20 parallel) ---")
	urls := make([]string, 20)
	for i := range urls {
		urls[i] = url
	}
	start = time.Now()
	results := FetchAll(ctx, goodClient, urls)
	fmt.Printf("20 concurrent requests: %v\n", time.Since(start))
	fmt.Printf("Results: %v\n", results[:5])
	fmt.Println()

	// 4. Context cancellation
	fmt.Println("--- Context Cancellation ---")
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Very slow
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	cancelCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start = time.Now()
	_, err := GoodRequest(cancelCtx, goodClient, slowServer.URL)
	fmt.Printf("Request with 100ms timeout to slow server: %v\n", time.Since(start))
	fmt.Printf("Error: %v\n", err)
	fmt.Println("✅ Request cancelled — didn't wait 5 seconds")
}
