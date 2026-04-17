// Ad Targeting Service - Walmart Platform
// Ad targeting and personalization engine.
//
// INTENTIONAL ISSUES (for demo):
// - Exposed PII in targeting segments (vulnerability)
// - No COPPA compliance check (vulnerability)
// - Missing authentication on bid endpoint (vulnerability)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
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

const maxImpressions = 10000

var (
	segments    []AdSegment
	impressions []map[string]interface{}
	impMutex    sync.Mutex
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

func addImpression(impression map[string]interface{}) {
	impMutex.Lock()
	defer impMutex.Unlock()

	impressions = append(impressions, impression)
	
	// Evict oldest entries if cache exceeds max size
	if len(impressions) > maxImpressions {
		impressions = impressions[len(impressions)-maxImpressions:]
	}
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

	impression := map[string]interface{}{
		"impression_id": fmt.Sprintf("IMP-%d", bidCount),
		"user_id":       req.UserID,
		"segment":       selectedSegment.SegmentID,
		"win_price":     winPrice,
		"timestamp":     time.Now().Unix(),
	}
	addImpression(impression)

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
		"behavioral_data":   map[string]interface{}{"purchase_history": "electronics, toys", "avg_order_value": 156.30},
		"predicted_ltv":     "$2,400",
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	impMutex.Lock()
	impCount := len(impressions)
	impMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_bids":        bidCount,
		"impressions_count": impCount,
		"active_segments":   len(segments),
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
	http.HandleFunc("/user/targeting", userTargetingHandler)
	http.HandleFunc("/metrics", metricsHandler)

	log.Printf("Ad Targeting Service listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
