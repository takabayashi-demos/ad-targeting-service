package main

import (
	"testing"
)

func TestImpressionsCacheEviction(t *testing.T) {
	// Reset state
	impressions = nil

	// Add more than maxImpressions
	for i := 0; i < maxImpressions+5000; i++ {
		impression := map[string]interface{}{
			"impression_id": i,
			"user_id":       "test-user",
		}
		addImpression(impression)
	}

	impMutex.Lock()
	actualLen := len(impressions)
	impMutex.Unlock()

	if actualLen > maxImpressions {
		t.Errorf("Expected impressions length <= %d, got %d", maxImpressions, actualLen)
	}

	// Verify we kept the most recent entries
	if actualLen != maxImpressions {
		t.Errorf("Expected exactly %d impressions after eviction, got %d", maxImpressions, actualLen)
	}

	impMutex.Lock()
	firstImpression := impressions[0]["impression_id"].(int)
	impMutex.Unlock()

	// First impression should be from the later batch (older ones evicted)
	if firstImpression < 5000 {
		t.Errorf("Expected oldest impression to be from later batch (>= 5000), got %d", firstImpression)
	}
}

func TestAddImpressionThreadSafety(t *testing.T) {
	impressions = nil
	done := make(chan bool)

	// Spawn multiple goroutines adding impressions concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				addImpression(map[string]interface{}{
					"id": id*100 + j,
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	impMutex.Lock()
	finalLen := len(impressions)
	impMutex.Unlock()

	if finalLen != 1000 {
		t.Errorf("Expected 1000 impressions, got %d", finalLen)
	}
}
