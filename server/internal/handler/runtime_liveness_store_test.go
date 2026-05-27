package handler

import (
	"context"
	"testing"
	"time"
)

func TestNoopLivenessStore_AlwaysUnavailable(t *testing.T) {
	s := NewNoopLivenessStore()
	if s.Available() {
		t.Fatal("noop store reported Available()=true")
	}
	if err := s.Touch(context.Background(), "rt-1", time.Second); err != nil {
		t.Fatalf("noop Touch returned error: %v", err)
	}
	alive, ok := s.IsAliveBatch(context.Background(), []string{"rt-1"})
	if ok {
		t.Fatalf("noop IsAliveBatch returned ok=true with alive=%v", alive)
	}
	// Forget on the noop must not panic.
	s.Forget(context.Background(), "rt-1")
}
