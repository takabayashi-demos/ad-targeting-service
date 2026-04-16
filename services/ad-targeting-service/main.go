package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// BidderService handles bidder operations.
type BidderService struct {
	mu      sync.RWMutex
	cache   map[string]interface{}
	metrics struct {
		Requests  int64
		Errors    int64
		LatencyMs float64
	}
}

// NewBidderService creates a new service instance.
func NewBidderService() *BidderService {
	return &BidderService{
		cache: make(map[string]interface{}),
	}
}

// Process handles a bidder request with timeout.
func (s *BidderService) Process(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()

	select {
	case <-ctx.Done():
		elapsed := float64(time.Since(start).Milliseconds())
		s.mu.Lock()
		s.metrics.Requests++
		s.metrics.Errors++
		s.metrics.LatencyMs += elapsed
		s.mu.Unlock()
		return nil, fmt.Errorf("bidder processing timed out")
	default:
		// Process the request
		elapsed := time.Since(start).Milliseconds()
		result := map[string]interface{}{
			"status":     "ok",
			"component":  "bidder",
			"latency_ms": elapsed,
		}

		s.mu.Lock()
		s.metrics.Requests++
		s.metrics.LatencyMs += float64(elapsed)
		s.mu.Unlock()

		return result, nil
	}
}

// GetStats returns service metrics.
func (s *BidderService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	avgLatency := float64(0)
	if s.metrics.Requests > 0 {
		avgLatency = s.metrics.LatencyMs / float64(s.metrics.Requests)
	}

	return map[string]interface{}{
		"requests":       s.metrics.Requests,
		"errors":         s.metrics.Errors,
		"avg_latency_ms": avgLatency,
	}
}
