package asn

import (
	"net/netip"
	"testing"
)

func TestNewASN(t *testing.T) {
	cache := NewASN()
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if cache.IPv4Tree() == nil {
		t.Error("expected non-nil IPv4Tree")
	}
	if cache.IPv6Tree() == nil {
		t.Error("expected non-nil IPv6Tree")
	}
}

func TestASNAdd(t *testing.T) {
	cache := NewASN()

	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("2001:db8::/32"),
	}

	cache.Add(15169, prefixes)

	if cache.Count() != 2 {
		t.Errorf("expected count 2, got %d", cache.Count())
	}
}

func TestASNContains(t *testing.T) {
	cache := NewASN()

	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("2001:db8::/32"),
	}
	cache.Add(15169, prefixes)

	// Test IPv4
	if !cache.Contains(netip.MustParseAddr("8.8.8.8")) {
		t.Error("expected to find 8.8.8.8")
	}
	if cache.Contains(netip.MustParseAddr("9.9.9.9")) {
		t.Error("expected not to find 9.9.9.9")
	}

	// Test IPv6
	if !cache.Contains(netip.MustParseAddr("2001:db8::1")) {
		t.Error("expected to find 2001:db8::1")
	}
	if !cache.Contains(netip.MustParseAddr("2001:db8:1::1")) {
		t.Error("expected to find 2001:db8:1::1 (included in 2001:db8::/32)")
	}
	if cache.Contains(netip.MustParseAddr("2001:db7::1")) {
		t.Error("expected not to find 2001:db7::1")
	}
}

func TestASNClone(t *testing.T) {
	cache := NewASN()
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
	}
	cache.Add(15169, prefixes)

	clone := cache.Clone()
	if clone == nil {
		t.Fatal("expected non-nil clone")
	}
	if clone.Count() != cache.Count() {
		t.Errorf("clone count %d != original count %d", clone.Count(), cache.Count())
	}
}

func TestDeduplicate(t *testing.T) {
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("9.9.9.0/24"),
	}

	result := Deduplicate(prefixes)
	if len(result) != 2 {
		t.Errorf("expected 2 unique prefixes, got %d", len(result))
	}
}

func TestValidatePrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		expected bool
	}{
		{"8.8.8.0/24", true},
		{"0.0.0.0/0", false},
		{"127.0.0.1/32", false},
		{"224.0.0.0/4", false},
	}

	for _, tt := range tests {
		prefix := netip.MustParsePrefix(tt.prefix)
		if got := ValidatePrefix(prefix); got != tt.expected {
			t.Errorf("ValidatePrefix(%s) = %v, want %v", tt.prefix, got, tt.expected)
		}
	}
}

func TestFilterInvalidPrefixes(t *testing.T) {
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("0.0.0.0/0"),
		netip.MustParsePrefix("127.0.0.1/32"),
	}

	result := FilterInvalidPrefixes(prefixes)
	if len(result) != 1 {
		t.Errorf("expected 1 valid prefix, got %d", len(result))
	}
}
