package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "UP" {
		t.Errorf("expected status UP, got %s", resp["status"])
	}
	if resp["service"] != serviceName {
		t.Errorf("expected service %s, got %s", serviceName, resp["service"])
	}
}

func TestBidHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/bid", nil)
	w := httptest.NewRecorder()

	bidHandler(w, req)

	if w.Code != statusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", statusMethodNotAllowed, w.Code)
	}
}

func TestBidHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/bid", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	bidHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestBidHandler_Success(t *testing.T) {
	bidReq := BidRequest{
		UserID:     "user-123",
		PageType:   "product",
		Categories: []string{"electronics"},
	}
	body, _ := json.Marshal(bidReq)

	req := httptest.NewRequest("POST", "/bid", bytes.NewReader(body))
	w := httptest.NewRecorder()

	bidHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp BidResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.BidID == "" {
		t.Error("expected non-empty bid_id")
	}
	if resp.Segment == "" {
		t.Error("expected non-empty segment")
	}
	if resp.WinPrice == "" {
		t.Error("expected non-empty win_price")
	}
}

func TestSegmentsHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/segments", nil)
	w := httptest.NewRecorder()

	segmentsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	total := int(resp["total"].(float64))
	if total != len(segments) {
		t.Errorf("expected %d segments, got %d", len(segments), total)
	}
}

func TestReadyHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "READY" {
		t.Errorf("expected status READY, got %s", resp["status"])
	}
}
