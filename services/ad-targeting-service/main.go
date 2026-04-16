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

const maxRequestBodyBytes = 1 << 20 // 1 MB

// BidderService handles bidder operations.
type BidderService struct {
	mu      sync.RWMutex
	cache   map[string]interface{}
	metrics struct {
		Requests  int64
		Errors    int64
		LatencyMs float64
	}
	rateLimiter *RateLimiter
}

// NewBidderService creates a new service instance.
func NewBidderService() *BidderService {
	return &BidderService{
		cache:       make(map[string]interface{}),
		rateLimiter: NewRateLimiter(50, 100, 5*time.Minute),
	}
}

// Process handles a bidder request with timeout.
func (s *BidderService) Process(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := validateBidderRequest(req); err != nil {
		s.mu.Lock()
		s.metrics.Errors++
		s.mu.Unlock()
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	start := time.Now()
	s.mu.Lock()
	s.metrics.Requests++
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		s.mu.Lock()
		s.metrics.Errors++
		s.mu.Unlock()
		return nil, fmt.Errorf("bidder processing timed out")
	default:
		result := map[string]interface{}{
			"status":     "ok",
			"component":  "bidder",
			"latency_ms": time.Since(start).Milliseconds(),
		}

		s.mu.Lock()
		s.metrics.LatencyMs += float64(time.Since(start).Milliseconds())
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

// ServeHTTP wires up the HTTP handlers with rate limiting and body size limits.
func (s *BidderService) ServeHTTP() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/bid", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		ip := clientIP(r)
		if !s.rateLimiter.Allow(ip) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON or body too large"}`, http.StatusBadRequest)
			return
		}

		result, err := s.Process(r.Context(), req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.GetStats())
	})

	return mux
}

func main() {
	svc := NewBidderService()
	server := &http.Server{
		Addr:         ":8080",
		Handler:      svc.ServeHTTP(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("ad-targeting-service starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
