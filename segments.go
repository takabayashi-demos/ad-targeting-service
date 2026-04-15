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

// TrackerService handles tracker operations.
type TrackerService struct {
	mu      sync.RWMutex
	cache   map[string]interface{}
	metrics struct {
		Requests  int64
		Errors    int64
		LatencyMs float64
	}
}

// NewTrackerService creates a new service instance.
func NewTrackerService() *TrackerService {
	return &TrackerService{
		cache: make(map[string]interface{}),
	}
}

// Process handles a tracker request with timeout.
func (s *TrackerService) Process(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	s.mu.Lock()
	s.metrics.Requests++
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		s.mu.Lock()
		s.metrics.Errors++
		s.mu.Unlock()
		return nil, fmt.Errorf("tracker processing timed out")
	default:
		// Process the request
		result := map[string]interface{}{
			"status":     "ok",
			"component":  "tracker",
			"latency_ms": time.Since(start).Milliseconds(),
		}

		s.mu.Lock()
		s.metrics.LatencyMs += float64(time.Since(start).Milliseconds())
		s.mu.Unlock()

		return result, nil
	}
}

// GetStats returns service metrics.
func (s *TrackerService) GetStats() map[string]interface{} {
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
