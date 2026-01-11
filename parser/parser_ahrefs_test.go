package parser

import (
	"net/netip"
	"strings"
	"testing"
)

func TestAhrefsParser(t *testing.T) {
	p := &AhrefsParser{}

	testCases := []struct {
		name     string
		input    string
		expected []netip.Prefix
	}{
		{
			name: "Ahrefs IP list format",
			input: `{
				"ips": [
					{"ip_address": "5.39.1.224"},
					{"ip_address": "5.39.1.225"},
					{"ip_address": "15.235.27.0"},
					{"ip_address": "15.235.27.1"}
				]
			}`,
			expected: []netip.Prefix{
				netip.MustParsePrefix("5.39.1.224/32"),
				netip.MustParsePrefix("5.39.1.225/32"),
				netip.MustParsePrefix("15.235.27.0/32"),
				netip.MustParsePrefix("15.235.27.1/32"),
			},
		},
		{
			name:     "Empty IP list",
			input:    `{"ips": []}`,
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

func TestIntegration_AhrefsBot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	p := &AhrefsParser{}
	data, err := fetchFromURL("https://api.ahrefs.com/v3/public/crawler-ips")
	if err != nil {
		t.Skipf("network unavailable, skipping: %v", err)
	}

	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(result) == 0 {
		t.Error("AhrefsBot IP list should not be empty")
	}

	// Verify all results are valid IPv4 prefixes
	for _, prefix := range result {
		if !prefix.Addr().Is4() {
			t.Errorf("expected IPv4 prefix, got %v", prefix)
		}
	}

	t.Logf("AhrefsBot: parsed %d prefixes", len(result))
}
