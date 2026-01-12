package knownbots

import (
	"net/netip"
	"testing"

	"github.com/cnlangzi/knownbots/asn"
)

func BenchmarkASNCache_Add(b *testing.B) {
	cache := NewASN()
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.4.0/24"),
		netip.MustParsePrefix("1.1.1.0/24"),
		netip.MustParsePrefix("1.0.0.0/24"),
		netip.MustParsePrefix("9.9.9.0/24"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Add(15169, prefixes)
	}
}

func BenchmarkASNCache_Contains(b *testing.B) {
	cache := NewASN()
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.4.0/24"),
		netip.MustParsePrefix("1.1.1.0/24"),
		netip.MustParsePrefix("1.0.0.0/24"),
		netip.MustParsePrefix("9.9.9.0/24"),
	}
	cache.Add(15169, prefixes)

	testIPs := []netip.Addr{
		netip.MustParseAddr("8.8.8.8"),
		netip.MustParseAddr("8.8.4.4"),
		netip.MustParseAddr("1.1.1.1"),
		netip.MustParseAddr("9.9.9.9"),
		netip.MustParseAddr("1.2.3.4"),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := testIPs[i%len(testIPs)]
			cache.Contains(ip)
			i++
		}
	})
}

func BenchmarkASNCache_Count(b *testing.B) {
	cache := NewASN()
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.4.0/24"),
		netip.MustParsePrefix("1.1.1.0/24"),
	}
	cache.Add(15169, prefixes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Count()
	}
}

func BenchmarkManager_VerifyIP(b *testing.B) {
	b.ReportAllocs()

	cache := NewASN()
	prefixes := map[int][]netip.Prefix{
		15169: {
			netip.MustParsePrefix("8.8.8.0/24"),
			netip.MustParsePrefix("8.8.4.0/24"),
		},
		13335: {
			netip.MustParsePrefix("1.1.1.0/24"),
			netip.MustParsePrefix("1.0.0.0/24"),
		},
		20940: {
			netip.MustParsePrefix("23.21.0.0/16"),
		},
	}
	for asn, ps := range prefixes {
		cache.Add(asn, ps)
	}

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"23.21.100.50",
		"9.9.9.9",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := testIPs[i%len(testIPs)]
			cache.Contains(netip.MustParseAddr(ip))
			i++
		}
	})
}

func BenchmarkDeduplicate(b *testing.B) {
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("1.1.1.0/24"),
		netip.MustParsePrefix("1.1.1.0/24"),
		netip.MustParsePrefix("9.9.9.0/24"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Deduplicate(prefixes)
	}
}

func BenchmarkValidatePrefix(b *testing.B) {
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("0.0.0.0/0"),
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("224.0.0.0/4"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range prefixes {
			asn.ValidatePrefix(p)
		}
	}
}
