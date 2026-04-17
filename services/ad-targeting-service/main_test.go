package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := NewRateLimiter(10, 5, time.Minute)

	for i := 0; i < 5; i++ {
		if !rl.Allow("10.0.0.1") {
			t.Fatalf("request %d should have been allowed within burst", i+1)
		}
	}

	if rl.Allow("10.0.0.1") {
		t.Fatal("request beyond burst should have been rejected")
	}
}

func TestRateLimiter_SeparatesClients(t *testing.T) {
	rl := NewRateLimiter(10, 2, time.Minute)

	rl.Allow("10.0.0.1")
	rl.Allow("10.0.0.1")

	if !rl.Allow("10.0.0.2") {
		t.Fatal("different IP should have its own bucket")
	}
}

func TestRateLimiter_ReplenishesTokens(t *testing.T) {
	rl := NewRateLimiter(1000, 1, time.Minute) // high rate so tokens replenish fast

	rl.Allow("10.0.0.1") // exhaust single token

	time.Sleep(10 * time.Millisecond) // wait for replenishment

	if !rl.Allow("10.0.0.1") {
		t.Fatal("token should have replenished")
	}
}

func TestValidation_NilRequest(t *testing.T) {
	err := validateBidderRequest(nil)
	if err == nil {
		t.Fatal("nil request should be rejected")
	}
}

func TestValidation_TooManyFields(t *testing.T) {
	req := make(map[string]interface{})
	for i := 0; i < 25; i++ {
		req[string(rune('a'+i))+"_field"] = "value"
	}
	err := validateBidderRequest(req)
	if err == nil {
		t.Fatal("request with >20 fields should be rejected")
	}
}

func TestValidation_InvalidPlacementID(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
	}{
		{"not a string", 12345},
		{"special chars", "drop'; SELECT--"},
		{"too long", strings.Repeat("a", 200)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := map[string]interface{}{"placement_id": tc.value}
			if err := validateBidderRequest(req); err == nil {
				t.Errorf("placement_id=%v should be rejected", tc.value)
			}
		})
	}
}

func TestValidation_ControlCharacters(t *testing.T) {
	req := map[string]interface{}{
		"user_agent": "Mozilla/5.0\r\nX-Injected: true",
	}
	if err := validateBidderRequest(req); err == nil {
		t.Fatal("control characters in field values should be rejected")
	}
}

func TestValidation_OversizedField(t *testing.T) {
	req := map[string]interface{}{
		"targeting": strings.Repeat("x", 2048),
	}
	if err := validateBidderRequest(req); err == nil {
		t.Fatal("oversized field should be rejected")
	}
}

func TestValidation_ValidRequest(t *testing.T) {
	req := map[string]interface{}{
		"placement_id": "homepage-banner-01",
		"campaign_id":  "camp_2024_q1",
		"ad_slots":     []interface{}{"slot1", "slot2"},
	}
	if err := validateBidderRequest(req); err != nil {
		t.Fatalf("valid request was rejected: %v", err)
	}
}

func TestValidation_AdSlotsTooMany(t *testing.T) {
	slots := make([]interface{}, 60)
	for i := range slots {
		slots[i] = "slot"
	}
	req := map[string]interface{}{"ad_slots": slots}
	if err := validateBidderRequest(req); err == nil {
		t.Fatal("too many ad_slots should be rejected")
	}
}

func TestProcess_ValidRequest(t *testing.T) {
	svc := NewBidderService()
	req := map[string]interface{}{
		"placement_id": "banner-01",
	}
	result, err := svc.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", result["status"])
	}
}

func TestProcess_InvalidRequest(t *testing.T) {
	svc := NewBidderService()
	req := map[string]interface{}{
		"placement_id": "invalid id with spaces!",
	}
	_, err := svc.Process(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid request")
	}
}

func TestHTTP_RateLimitResponse(t *testing.T) {
	svc := NewBidderService()
	// Override with a very small limiter
	svc.rateLimiter = NewRateLimiter(0.1, 1, time.Minute)

	handler := svc.ServeHTTP()

	body, _ := json.Marshal(map[string]interface{}{"placement_id": "test-01"})

	// First request should succeed
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first request should succeed, got %d", w.Code)
	}

	// Second request should be rate limited
	req = httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be rate limited, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("rate limited response should include Retry-After header")
	}
}

func TestHTTP_MethodNotAllowed(t *testing.T) {
	svc := NewBidderService()
	handler := svc.ServeHTTP()

	req := httptest.NewRequest(http.MethodGet, "/bid", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /bid should return 405, got %d", w.Code)
	}
}

func TestHTTP_OversizedBody(t *testing.T) {
	svc := NewBidderService()
	handler := svc.ServeHTTP()

	// Send a body larger than 1MB
	huge := strings.Repeat("x", 2<<20)
	req := httptest.NewRequest(http.MethodPost, "/bid", strings.NewReader(huge))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("oversized body should return 400, got %d", w.Code)
	}
}
