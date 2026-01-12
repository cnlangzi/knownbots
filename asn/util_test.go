package asn

import (
	"net/netip"
	"testing"
)

func TestDistinct(t *testing.T) {
	tests := []struct {
		name     string
		input    []netip.Prefix
		expected []netip.Prefix
	}{
		{
			name: "filters invalid prefixes",
			input: []netip.Prefix{
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("0.0.0.0/0"),
				netip.MustParsePrefix("127.0.0.1/32"),
				netip.MustParsePrefix("1.1.1.0/24"),
			},
			expected: []netip.Prefix{
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("1.1.1.0/24"),
			},
		},
		{
			name: "removes duplicates",
			input: []netip.Prefix{
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("1.1.1.0/24"),
				netip.MustParsePrefix("8.8.8.0/24"),
			},
			expected: []netip.Prefix{
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("1.1.1.0/24"),
			},
		},
		{
			name: "filters and deduplicates together",
			input: []netip.Prefix{
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("0.0.0.0/0"),
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("127.0.0.1/32"),
				netip.MustParsePrefix("1.1.1.0/24"),
				netip.MustParsePrefix("224.0.0.0/4"),
				netip.MustParsePrefix("1.1.1.0/24"),
			},
			expected: []netip.Prefix{
				netip.MustParsePrefix("8.8.8.0/24"),
				netip.MustParsePrefix("1.1.1.0/24"),
			},
		},
		{
			name:     "handles empty input",
			input:    []netip.Prefix{},
			expected: []netip.Prefix{},
		},
		{
			name: "filters all invalid",
			input: []netip.Prefix{
				netip.MustParsePrefix("0.0.0.0/0"),
				netip.MustParsePrefix("127.0.0.1/32"),
				netip.MustParsePrefix("224.0.0.0/4"),
			},
			expected: []netip.Prefix{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Distinct(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d prefixes, got %d", len(tt.expected), len(result))
			}
			for i, prefix := range result {
				if i >= len(tt.expected) || prefix != tt.expected[i] {
					t.Errorf("result[%d] = %s, want %s", i, prefix, tt.expected[i])
				}
			}
		})
	}
}

func BenchmarkDistinct(b *testing.B) {
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("8.8.8.0/24"),
		netip.MustParsePrefix("0.0.0.0/0"),
		netip.MustParsePrefix("1.1.1.0/24"),
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("1.1.1.0/24"),
		netip.MustParsePrefix("9.9.9.0/24"),
		netip.MustParsePrefix("224.0.0.0/4"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Distinct(prefixes)
	}
}
