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

func TestTxtParserEdgeCases(t *testing.T) {
	p := &TxtParser{}

	testCases := []struct {
		name     string
		input    string
		expected []netip.Prefix
	}{
		{
			name:  "Leading and trailing whitespace",
			input: "  1.2.3.4/24  \n\t5.6.7.8/24\t\n  9.10.11.12/32  ",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
				netip.MustParsePrefix("9.10.11.12/32"),
			},
		},
		{
			name:  "Multiple blank lines",
			input: "1.2.3.4/24\n\n\n5.6.7.8/24\n\n\n9.10.11.12/32",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
				netip.MustParsePrefix("9.10.11.12/32"),
			},
		},
		{
			name:  "Whitespace-only lines",
			input: "1.2.3.4/24\n   \n\t\n  \t  \n5.6.7.8/24",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
			},
		},
		{
			name:  "Invalid CIDR lines are ignored",
			input: "1.2.3.4/24\ninvalid-cidr\n5.6.7.8/24\nnot a valid prefix\n9.10.11.12/32",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
				netip.MustParsePrefix("9.10.11.12/32"),
			},
		},
		{
			name:  "Invalid IP addresses are ignored",
			input: "1.2.3.4/24\n999.999.999.999\n5.6.7.8/24\ninvalid.ip.address\n9.10.11.12/32",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
				netip.MustParsePrefix("9.10.11.12/32"),
			},
		},
		{
			name:  "Duplicate entries are allowed",
			input: "1.2.3.4/24\n1.2.3.4/24\n1.2.3.4/24",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("1.2.3.4/24"),
			},
		},
		{
			name:  "Mixed IPv4 and IPv6",
			input: "1.2.3.4/24\n2001:db8::/32\n5.6.7.8/24\n2001:db8:1::/48",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("2001:db8::/32"),
				netip.MustParsePrefix("5.6.7.8/24"),
				netip.MustParsePrefix("2001:db8:1::/48"),
			},
		},
		{
			name:  "Individual IPs are converted to /32 or /128",
			input: "1.2.3.4\n5.6.7.8\n2001:db8::1",
			expected: []netip.Prefix{
				netip.PrefixFrom(netip.MustParseAddr("1.2.3.4"), 32),
				netip.PrefixFrom(netip.MustParseAddr("5.6.7.8"), 32),
				netip.PrefixFrom(netip.MustParseAddr("2001:db8::1"), 128),
			},
		},
		{
			name:  "Comments with various prefixes",
			input: "# This is a comment\n1.2.3.4/24\n# Another comment starting with #\n5.6.7.8/24",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
			},
		},
		{
			name:  "Malformed lines with extra spaces",
			input: "  1.2.3.4/24  \n  5.6.7.8/24  \n",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
			},
		},
		{
			name:  "IPv4-mapped IPv6 addresses",
			input: "::ffff:1.2.3.4/128",
			expected: []netip.Prefix{
				netip.MustParsePrefix("::ffff:1.2.3.4/128"),
			},
		},
		{
			name:  "Invalid prefix lengths are ignored",
			input: "1.2.3.4/33\n1.2.3.4/-1\n5.6.7.8/24",
			expected: []netip.Prefix{
				netip.MustParsePrefix("5.6.7.8/24"),
			},
		},
		{
			name:  "Lines with only special characters",
			input: "1.2.3.4/24\n---\n5.6.7.8/24\n---\n9.10.11.12/32",
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.4/24"),
				netip.MustParsePrefix("5.6.7.8/24"),
				netip.MustParsePrefix("9.10.11.12/32"),
			},
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

func TestIntegration_Cloudflare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &TxtParser{}

	// Fetch IPv4 addresses
	data, err := fetchFromURL("https://www.cloudflare.com/ips-v4/")
	if err != nil {
		t.Fatalf("failed to fetch IPv4: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("Cloudflare IPv4 list should not be empty")
	}
	t.Logf("Cloudflare IPv4: parsed %d prefixes", len(result))

	// Verify all results are valid IPv4 prefixes
	for _, prefix := range result {
		if !prefix.Addr().Is4() {
			t.Errorf("expected IPv4 prefix, got %v", prefix)
		}
	}
}
