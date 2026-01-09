package parser

import (
	"net/netip"
	"strings"
	"testing"
)

func TestGoogleParser(t *testing.T) {
	p := &GoogleParser{}

	testCases := []struct {
		name     string
		input    string
		expected []netip.Prefix
	}{
		{
			name: "Googlebot IPv4 and IPv6 prefixes",
			input: `{
				"prefixes": [
					{"ipv4Prefix": "66.249.64.0/19", "ipv6Prefix": "2001:db8::/32"},
					{"ipv4Prefix": "66.249.90.0/24", "ipv6Prefix": ""},
					{"ipv4Prefix": "", "ipv6Prefix": "2600:1900::/28"}
				]
			}`,
			expected: []netip.Prefix{
				netip.MustParsePrefix("66.249.64.0/19"),
				netip.MustParsePrefix("2001:db8::/32"),
				netip.MustParsePrefix("66.249.90.0/24"),
				netip.MustParsePrefix("2600:1900::/28"),
			},
		},
		{
			name: "Special crawlers format",
			input: `{
				"prefixes": [
					{"ipv4Prefix": "66.249.64.0/19"},
					{"ipv4Prefix": "142.250.0.0/15"}
				]
			}`,
			expected: []netip.Prefix{
				netip.MustParsePrefix("66.249.64.0/19"),
				netip.MustParsePrefix("142.250.0.0/15"),
			},
		},
		{
			name:     "Empty prefixes",
			input:    `{"prefixes": []}`,
			expected: []netip.Prefix{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d prefixes, got %d", len(tc.expected), len(result))
			}

			for i, exp := range tc.expected {
				if result[i] != exp {
					t.Errorf("expected prefix %d to be %v, got %v", i, exp, result[i])
				}
			}
		})
	}
}
