package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testServer *httptest.Server

func init() {
	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","data":{"id":"123","name":"test"}}`))
	}))
}

// --- Connection Reuse Benchmarks ---

func BenchmarkBadClientNoBodyClose(b *testing.B) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(testServer.URL)
		if err != nil {
			b.Fatal(err)
		}
		// ❌ Not closing/draining body
		_ = resp.StatusCode
	}
}

func BenchmarkGoodClientWithBodyDrain(b *testing.B) {
	client := GoodClient()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GoodRequest(ctx, client, testServer.URL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Transport Configuration Benchmarks ---

func BenchmarkDefaultTransport(b *testing.B) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkTunedTransport(b *testing.B) {
	client := GoodClient()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// --- Concurrent Benchmarks ---

func BenchmarkSequential10Requests(b *testing.B) {
	client := GoodClient()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			GoodRequest(ctx, client, testServer.URL)
		}
	}
}

func BenchmarkConcurrent10Requests(b *testing.B) {
	client := GoodClient()
	ctx := context.Background()
	urls := make([]string, 10)
	for i := range urls {
		urls[i] = testServer.URL
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		FetchAll(ctx, client, urls)
	}
}

// --- New Client Per Request vs Shared ---

func BenchmarkNewClientPerRequest(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkSharedClient(b *testing.B) {
	client := GoodClient()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
