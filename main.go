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

const (
	// Service metadata
	serviceName    = "ad-targeting-service"
	serviceVersion = "1.4.2"

	// Bidding parameters
	minBidLatencyMs = 10
	maxBidLatencyMs = 30
	bidPriceFloor   = 0.7
	bidPriceCeiling = 1.0

	// HTTP status codes
	statusMethodNotAllowed = 405
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

type BidResponse struct {
	BidID    string `json:"bid_id"`
	AdUnit   string `json:"ad_unit"`
	Creative string `json:"creative"`
	WinPrice string `json:"win_price"`
	Segment  string `json:"segment"`
}

var (
	segments []AdSegment
	// ❌ BUG: Memory leak - impressions grow forever, never cleaned
	impressions []map[string]interface{}
	bidCount    int64
	logger      *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "[ad-targeting] ", log.LstdFlags|log.Lshortfile)

	segments = []AdSegment{
		{SegmentID: "SEG-001", Name: "High-Value Electronics Shoppers", Users: 2500000, Criteria: []string{"purchased_electronics > 3", "avg_order > $200"}, BidPrice: 4.50},
		{SegmentID: "SEG-002", Name: "Young Parents 25-35", Users: 1800000, Criteria: []string{"age_range:25-35", "has_children:true", "baby_products > 0"}, BidPrice: 3.20},
		{SegmentID: "SEG-003", Name: "Budget Conscious Grocery", Users: 5000000, Criteria: []string{"coupon_usage > 5", "grocery_frequency:weekly"}, BidPrice: 1.80},
		{SegmentID: "SEG-004", Name: "Premium Brand Loyalists", Users: 800000, Criteria: []string{"brand_affinity:premium", "return_rate < 5%"}, BidPrice: 6.00},
		// ❌ VULNERABILITY: Age-based targeting without COPPA check
		{SegmentID: "SEG-005", Name: "Teen Gamers 13-17", Users: 900000, Criteria: []string{"age_range:13-17", "gaming_purchases > 2"}, BidPrice: 2.50},
	}
}

func respondJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	respondJSON(w, map[string]string{"error": message})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]string{
		"status":  "UP",
		"service": serviceName,
		"version": serviceVersion,
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]string{"status": "READY"})
}

func segmentsHandler(w http.ResponseWriter, r *http.Request) {
	logger.Printf("GET /segments - returning %d segments", len(segments))
	respondJSON(w, map[string]interface{}{
		"segments": segments,
		"total":    len(segments),
	})
}

func bidHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		logger.Printf("Invalid method %s for /bid", r.Method)
		respondError(w, statusMethodNotAllowed, "Method not allowed")
		return
	}

	// ❌ VULNERABILITY: No authentication on bid endpoint
	var req BidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("Failed to decode bid request: %v", err)
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	bidCount++

	// Simulate real-time bidding with latency
	latency := time.Duration(rand.Intn(maxBidLatencyMs-minBidLatencyMs)+minBidLatencyMs) * time.Millisecond
	time.Sleep(latency)

	// Select matching segment
	selectedSegment := segments[rand.Intn(len(segments))]
	winPrice := selectedSegment.BidPrice * (bidPriceFloor + rand.Float64()*(bidPriceCeiling-bidPriceFloor))

	// ❌ BUG: Memory leak - impressions accumulate without limit
	impression := map[string]interface{}{
		"impression_id": fmt.Sprintf("IMP-%d", bidCount),
		"user_id":       req.UserID,
		"segment":       selectedSegment.SegmentID,
		"win_price":     winPrice,
		"timestamp":     time.Now().Unix(),
	}
	impressions = append(impressions, impression)

	logger.Printf("Bid %d: user=%s segment=%s price=%.2f latency=%dms",
		bidCount, req.UserID, selectedSegment.SegmentID, winPrice, latency.Milliseconds())

	response := BidResponse{
		BidID:    fmt.Sprintf("BID-%d", bidCount),
		AdUnit:   "banner_728x90",
		Creative: fmt.Sprintf("https://ads.walmart.com/creatives/%s.png", selectedSegment.SegmentID),
		WinPrice: fmt.Sprintf("%.2f", winPrice),
		Segment:  selectedSegment.Name,
	}

	respondJSON(w, response)
}

// ❌ VULNERABILITY: Exposes user targeting data without auth
func userTargetingHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	logger.Printf("GET /user-targeting?user_id=%s", userID)

	respondJSON(w, map[string]interface{}{
		"user_id":          userID,
		"matched_segments": segments[:3],
		"behavioral_data": map[string]interface{}{
			"last_visit":     time.Now().Add(-2 * time.Hour).Unix(),
			"page_views":     rand.Intn(50) + 10,
			"purchase_count": rand.Intn(10),
		},
	})
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

	logger.Printf("Starting %s v%s on port %s", serviceName, serviceVersion, port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}
