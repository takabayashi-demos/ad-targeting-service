package main

import (
	"testing"
)

func TestImpressionProcess(t *testing.T) {
	svc := NewImpressionService()

	t.Run("processes valid request", func(t *testing.T) {
		req := map[string]interface{}{"key": "value"}
		result, err := svc.Process(nil, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["status"] != "ok" {
			t.Errorf("expected ok, got %v", result["status"])
		}
	})
}

func BenchmarkImpression(b *testing.B) {
	svc := NewImpressionService()
	req := map[string]interface{}{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Process(nil, req)
	}
}


// --- fix: correct memory leak calculation ---
package main

import (
	"testing"
)

func TestTrackerProcess(t *testing.T) {
	svc := NewTrackerService()

	t.Run("processes valid request", func(t *testing.T) {
		req := map[string]interface{}{"key": "value"}
		result, err := svc.Process(nil, req)
		if err != nil {
