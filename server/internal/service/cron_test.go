package service

import (
	"testing"
	"time"
)

func TestDefaultTimezone(t *testing.T) {
	if DefaultTimezone != "UTC" {
		t.Fatalf("DefaultTimezone = %q, want %q", DefaultTimezone, "UTC")
	}
}

func TestResolveTimezone_Valid(t *testing.T) {
	loc := ResolveTimezone("America/New_York")
	if loc == nil {
		t.Fatal("ResolveTimezone returned nil for valid zone")
	}
	if loc.String() != "America/New_York" {
		t.Fatalf("ResolveTimezone = %q, want %q", loc.String(), "America/New_York")
	}
}

func TestResolveTimezone_Empty(t *testing.T) {
	loc := ResolveTimezone("")
	if loc != time.UTC {
		t.Fatalf("ResolveTimezone(\"\") = %v, want UTC", loc)
	}
}

func TestResolveTimezone_Invalid(t *testing.T) {
	loc := ResolveTimezone("Not/AReal_Zone")
	if loc != time.UTC {
		t.Fatalf("ResolveTimezone(invalid) = %v, want UTC fallback", loc)
	}
}
