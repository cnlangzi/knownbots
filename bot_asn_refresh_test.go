package knownbots

import (
	"net/netip"
	"sync/atomic"
	"testing"
)

func TestRefreshASNRebuildsCache(t *testing.T) {
	bot := &Bot{
		Name: "testbot",
		ASN:  []int{15169, 13335},
	}

	initialPrefixesAS15169 := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.4.0/24"),
		netip.MustParsePrefix("2001:db8::/32"),
	}

	initialPrefixesAS13335 := []netip.Prefix{
		netip.MustParsePrefix("1.1.1.0/24"),
	}

	bot.asns = &atomic.Pointer[ASN]{}
	initialCache := NewASN()
	initialCache.Add(15169, initialPrefixesAS15169)
	initialCache.Add(13335, initialPrefixesAS13335)
	bot.asns.Store(initialCache)

	cache := bot.asns.Load()
	if !cache.Contains(netip.MustParseAddr("8.8.8.8")) {
		t.Error("Expected 8.8.8.8 to be in initial ASN cache")
	}
	if !cache.Contains(netip.MustParseAddr("8.8.4.4")) {
		t.Error("Expected 8.8.4.4 to be in initial ASN cache")
	}
	if !cache.Contains(netip.MustParseAddr("2001:db8::1")) {
		t.Error("Expected 2001:db8::1 to be in initial ASN cache")
	}
	if !cache.Contains(netip.MustParseAddr("1.1.1.1")) {
		t.Error("Expected 1.1.1.1 to be in initial ASN cache")
	}

	initialCount := cache.Count()
	if initialCount != 4 {
		t.Errorf("Expected 4 prefixes initially, got %d", initialCount)
	}

	newPrefixesAS15169 := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
	}

	newCache := NewASN()
	newCache.Add(15169, newPrefixesAS15169)
	newCache.Add(13335, initialPrefixesAS13335)
	bot.asns.Store(newCache)

	cache = bot.asns.Load()
	if !cache.Contains(netip.MustParseAddr("8.8.8.8")) {
		t.Error("Expected 8.8.8.8 to still be in ASN cache after rebuild")
	}

	if cache.Contains(netip.MustParseAddr("8.8.4.4")) {
		t.Error("Expected 8.8.4.4 to be REMOVED from ASN cache after rebuild")
	}
	if cache.Contains(netip.MustParseAddr("2001:db8::1")) {
		t.Error("Expected 2001:db8::1 to be REMOVED from ASN cache after rebuild")
	}

	if !cache.Contains(netip.MustParseAddr("1.1.1.1")) {
		t.Error("Expected 1.1.1.1 to remain in ASN cache (AS13335 unchanged)")
	}

	newCount := cache.Count()
	if newCount != 2 {
		t.Errorf("Expected 2 prefixes after rebuild, got %d", newCount)
	}
}
