// Ad Targeting Service - Walmart Platform
// Ad targeting and personalization engine.
//
// INTENTIONAL ISSUES (for demo):
// - Exposed PII in targeting segments (vulnerability)
// - No COPPA compliance check (vulnerability)
// - Memory leak in audience cache (bug)
// - Missing authentication on bid endpoint (vulnerability)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// HTTP status codes
const (
	StatusMethodNotAllowed = 405
)

// Bid timing configuration
const (
	MinBidLatencyMs = 10
	MaxBidLatencyMs = 30
)

// Bid price multipliers
const (
	MinBidMultiplier   = 0.7
	MaxBidMultiplier   = 1.0
	BidMultiplierRange = MaxBidMultiplier - MinBidMultiplier
)

// Ad creative configuration
const (
	DefaultAdUnit       = "banner_728x90"
	CreativeURLTemplate = "https://ads.walmart.com/creatives/%s.png"
)

type AdSegment struct {
	SegmentID string   `json:"segment_id"`
	Name      string   `json:"name"`
	Users     int      `json:"user_count"`
	Criteria  []string `json:"criteria"`
	BidPrice  float64  `json:"bid_price_cpm"`
}

type BidRequest struct {
	UserID     string   `json:"user_id"`
	PageType   string   `json:"page_type"`
	Categories []string `json:"categories"`
}

var (
	segments []AdSegment
	// ❌ BUG: Memory leak - impressions grow forever, never cleaned
	impressions []map[string]interface{}
	bidCount    int64
)

func init() {
	segments = []AdSegment{
		{SegmentID: "SEG-001", Name: "High-Value Electronics Shoppers", Users: 2500000, Criteria: []string{"purchased_electronics > 3", "avg_order > $200"}, BidPrice: 4.50},
		{SegmentID: "SEG-002", Name: "Young Parents 25-35", Users: 1800000, Criteria: []string{"age_range:25-35", "has_children:true", "baby_products > 0"}, BidPrice: 3.20},
		{SegmentID: "SEG-003", Name: "Budget Conscious Grocery", Users: 5000000, Criteria: []string{"coupon_usage > 5", "grocery_frequency:weekly"}, BidPrice: 1.80},
		{SegmentID: "SEG-004", Name: "Premium Brand Loyalists", Users: 800000, Criteria: []string{"brand_affinity:premium", "return_rate < 5%"}, BidPrice: 6.00},
		// ❌ VULNERABILITY: Age-based targeting without COPPA check
		{SegmentID: "SEG-005", Name: "Teen Gamers 13-17", Users: 900000, Criteria: []string{"age_range:13-17", "gaming_purchases > 2"}, BidPrice: 2.50},
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "UP", "service": "ad-targeting-service", "version": "1.4.2",
	}); err != nil {
		log.Printf("ERROR: Failed to encode health response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "READY"}); err != nil {
		log.Printf("ERROR: Failed to encode ready response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func segmentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"segments": segments,
		"total":    len(segments),
	}); err != nil {
		log.Printf("ERROR: Failed to encode segments response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func bidHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", StatusMethodNotAllowed)
		return
	}

	// ❌ VULNERABILITY: No authentication on bid endpoint
	var req BidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR: Failed to decode bid request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bidCount++
	log.Printf("INFO: Processing bid request %d for user %s", bidCount, req.UserID)

	// Simulate real-time bidding with latency
	latency := time.Duration(rand.Intn(MaxBidLatencyMs-MinBidLatencyMs)+MinBidLatencyMs) * time.Millisecond
	time.Sleep(latency)

	// Select matching segment and calculate win price
	selectedSegment := selectSegment()
	winPrice := calculateWinPrice(selectedSegment.BidPrice)

	// ❌ BUG: Memory leak - impressions accumulate without limit
	recordImpression(bidCount, req.UserID, selectedSegment.SegmentID, winPrice)

	response := map[string]interface{}{
		"bid_id":    fmt.Sprintf("BID-%d", bidCount),
		"ad_unit":   DefaultAdUnit,
		"creative":  fmt.Sprintf(CreativeURLTemplate, selectedSegment.SegmentID),
		"win_price": fmt.Sprintf("%.2f", winPrice),
		"segment":   selectedSegment.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode bid response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// selectSegment returns a randomly selected ad segment
func selectSegment() AdSegment {
	return segments[rand.Intn(len(segments))]
}

// calculateWinPrice computes the final bid price with random variance
func calculateWinPrice(basePrice float64) float64 {
	multiplier := MinBidMultiplier + rand.Float64()*BidMultiplierRange
	return basePrice * multiplier
}

// recordImpression stores impression data for analytics
func recordImpression(bidID int64, userID, segmentID string, winPrice float64) {
	impression := map[string]interface{}{
		"impression_id": fmt.Sprintf("IMP-%d", bidID),
		"user_id":       userID,
		"segment":       segmentID,
		"win_price":     winPrice,
		"timestamp":     time.Now().Unix(),
	}
	impressions = append(impressions, impression)
}

// ❌ VULNERABILITY: Exposes user targeting data without auth
func userTargetingHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	response := map[string]interface{}{
		"user_id":          userID,
		"matched_segments": segments[:3],
		"behavioral_data": map[string]interface{}{
			"page_views": rand.Intn(100),
			"cart_adds":  rand.Intn(20),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode user targeting response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)
	http.HandleFunc("/segments", segmentsHandler)
	http.HandleFunc("/bid", bidHandler)
	http.HandleFunc("/user-targeting", userTargetingHandler)

	log.Printf("INFO: Ad Targeting Service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("FATAL: Server failed to start: %v", err)
	}
}
