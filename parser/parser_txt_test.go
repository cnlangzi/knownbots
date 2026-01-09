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

func TestTxtParserWithRealFormat(t *testing.T) {
	p := &TxtParser{}
	testInput := "172.217.0.0/16\n142.250.0.0/15\n192.178.0.0/15"
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 prefixes, got %d", len(result))
	}
}

func TestIntegration_UptimeRobot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &TxtParser{}
	data, err := fetchFromURL("https://uptimerobot.com/inc/files/ips/IPv4.txt")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("UptimeRobot IP list should not be empty")
	}
	t.Logf("UptimeRobot: parsed %d prefixes", len(result))
}
