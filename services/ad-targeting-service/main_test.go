package main

import (
	"context"
	"sync"
	"testing"
)

func TestProcessMetricsConsistency(t *testing.T) {
	svc := NewBidderService()
	const goroutines = 50
	const requestsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				_, _ = svc.Process(context.Background(), map[string]interface{}{"id": "test"})
			}
		}()
	}

	wg.Wait()

	stats := svc.GetStats()
	totalRequests := stats["requests"].(int64)
	totalErrors := stats["errors"].(int64)
	avgLatency := stats["avg_latency_ms"].(float64)

	expected := int64(goroutines * requestsPerGoroutine)
	if totalRequests != expected {
		t.Errorf("expected %d requests, got %d", expected, totalRequests)
	}

	if totalErrors != 0 {
		t.Errorf("expected 0 errors, got %d", totalErrors)
	}

	if avgLatency < 0 {
		t.Errorf("avg_latency_ms should be non-negative, got %f", avgLatency)
	}
}

func TestProcessMetricsNotSkewedDuringConcurrentReads(t *testing.T) {
	svc := NewBidderService()
	const writers = 20
	const readers = 10
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	// Writers: send requests concurrently
	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = svc.Process(context.Background(), map[string]interface{}{"id": "test"})
			}
		}()
	}

	// Readers: read stats concurrently and verify consistency
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				stats := svc.GetStats()
				reqs := stats["requests"].(int64)
				errs := stats["errors"].(int64)

				if errs > reqs {
					t.Errorf("errors (%d) exceeded requests (%d) — metrics are inconsistent", errs, reqs)
				}
			}
		}()
	}

	wg.Wait()
}

func TestProcessWithCancelledContext(t *testing.T) {
	svc := NewBidderService()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := svc.Process(ctx, map[string]interface{}{"id": "test"})

	// Whether it errors or not depends on select scheduling,
	// but metrics must always be consistent afterward.
	stats := svc.GetStats()
	reqs := stats["requests"].(int64)
	errs := stats["errors"].(int64)

	if reqs != 1 {
		t.Errorf("expected 1 request, got %d", reqs)
	}

	if err != nil && errs != 1 {
		t.Errorf("process returned error but errors metric is %d, expected 1", errs)
	}

	if err == nil && errs != 0 {
		t.Errorf("process succeeded but errors metric is %d, expected 0", errs)
	}
}
