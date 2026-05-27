package http_client_optimization

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

// BadClient uses http.DefaultClient — no timeouts, default transport.
func BadClient() *http.Client {
	return http.DefaultClient
}

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

// BadRequest doesn't close body, no timeout, no context.
func BadRequest(client *http.Client, url string) (int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	// BAD: body not read/closed — connection can't be reused!
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

	// GOOD: drain body to allow connection reuse
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

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
