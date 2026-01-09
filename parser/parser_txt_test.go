package parser

import (
	"net/netip"
	"strings"
	"testing"
)

func TestTxtParser(t *testing.T) {
	p := &TxtParser{}

	testCases := []struct {
		name     string
		input    string
		expected []netip.Prefix
	}{
		{
			name:  "Simple line-by-line",
			input: "1.2.3.4/24\n5.6.7.8/24\n9.10.11.12/32",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
				netip.MustParsePrefix("9.10.11.12/32"),
			},
		},
		{
			name:  "With comments and empty lines",
			input: "# Comment line\n1.2.3.4/24\n\n5.6.7.8/24\n# Another comment",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
			},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []netip.Prefix{},
		},
		{
			name:     "Only comments",
			input:    "# Just a comment\n# Another comment",
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
