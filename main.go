package main

import (
	"testing"
)

func TestSegmentProcess(t *testing.T) {
	svc := NewSegmentService()

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

func BenchmarkSegment(b *testing.B) {
	svc := NewSegmentService()
	req := map[string]interface{}{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Process(nil, req)
	}
}
