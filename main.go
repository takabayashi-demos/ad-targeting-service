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

type AdSegment struct {
	SegmentID string   `json:"segment_id"`
	Name      string   `json:"name"`
	Users     int      `json:"user_count"`
	Criteria  []string `json:"criteria"`
	BidPrice  float64  `json:"bid_price_cpm"`
}

type BidRequest struct {
	UserID     string `json:"user_id"`
	PageType   string `json:"page_type"`
	Categories []string `json:"categories"`
}

var (
	segments    []AdSegment
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
	json.NewEncoder(w).Encode(map[string]string{
		"status": "UP", "service": "ad-targeting-service", "version": "1.4.2",
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "READY"})
}

func segmentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"segments": segments,
		"total":    len(segments),
	})
}

func bidHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// ❌ VULNERABILITY: No authentication on bid endpoint
	var req BidRequest
	json.NewDecoder(r.Body).Decode(&req)

	bidCount++

	// Simulate real-time bidding with latency
	time.Sleep(time.Duration(rand.Intn(30)+10) * time.Millisecond)

	// Select matching segment
	selectedSegment := segments[rand.Intn(len(segments))]
	winPrice := selectedSegment.BidPrice * (0.7 + rand.Float64()*0.3)

	// ❌ BUG: Memory leak - impressions accumulate without limit
	impression := map[string]interface{}{
		"impression_id": fmt.Sprintf("IMP-%d", bidCount),
		"user_id":       req.UserID,
		"segment":       selectedSegment.SegmentID,
		"win_price":     winPrice,
		"timestamp":     time.Now().Unix(),
	}
	impressions = append(impressions, impression)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bid_id":    fmt.Sprintf("BID-%d", bidCount),
		"ad_unit":   "banner_728x90",
		"creative":  fmt.Sprintf("https://ads.walmart.com/creatives/%s.png", selectedSegment.SegmentID),
		"win_price": fmt.Sprintf("%.2f", winPrice),
		"segment":   selectedSegment.Name,
	})
}

// ❌ VULNERABILITY: Exposes user targeting data without auth
func userTargetingHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":           userID,
		"matched_segments":  segments[:3],
		"behavioral_data":   map[string]interface{}{"page_views": 47, "cart_adds": 12, "purchases": 3},
		"predicted_ltv":     "$2,340",
		"churn_probability": 0.15,
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `# HELP ad_bids_total Total bid requests processed
# TYPE ad_bids_total counter
ad_bids_total %d
# HELP ad_impressions_stored Impressions in memory (leak indicator)
# TYPE ad_impressions_stored gauge
ad_impressions_stored %d
# HELP ad_service_up Service health
# TYPE ad_service_up gauge
ad_service_up 1
`, bidCount, len(impressions))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)
	http.HandleFunc("/api/v1/segments", segmentsHandler)
	http.HandleFunc("/api/v1/bid", bidHandler)
	http.HandleFunc("/api/v1/targeting", userTargetingHandler)
	http.HandleFunc("/metrics", metricsHandler)

	log.Printf("ad-targeting-service starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
// Age gate check
// Frequency cap
// Buffer limit
// Context targeting
