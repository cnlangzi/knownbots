package parser

import (
	"net/netip"
	"strings"
	"testing"
)

func TestAmazonParser(t *testing.T) {
	p := &AmazonParser{}

	testCases := []struct {
		name     string
		input    string
		expected []netip.Prefix
	}{
		{
			name: "AmazonBot HTML with embedded JSON",
			input: `<!doctype html>
<html>
<body>
<pre>
{
  "creationTime": "2025-11-04T16:12:45.697663+00:00",
  "prefixes": [
    {
      "ipv4Prefix": "100.24.149.244"
    },
    {
      "ipv4Prefix": "100.24.167.60"
    },
    {
      "ipv4Prefix": "100.25.120.246"
    },
    {
      "ipv4Prefix": "18.204.89.56"
    },
    {
      "ipv4Prefix": "3.89.170.186"
    },
    {
      "ipv4Prefix": "54.144.185.255"
    }
  ]
}
</pre>
</body>
</html>`,
			expected: []netip.Prefix{
				netip.MustParsePrefix("100.24.149.244/32"),
				netip.MustParsePrefix("100.24.167.60/32"),
				netip.MustParsePrefix("100.25.120.246/32"),
				netip.MustParsePrefix("18.204.89.56/32"),
				netip.MustParsePrefix("3.89.170.186/32"),
				netip.MustParsePrefix("54.144.185.255/32"),
			},
		},
		{
			name:  "Fallback: plain text IPs",
			input: `<html><body><p>IPs: 100.24.149.244, 18.204.89.56, 3.89.170.186</p></body></html>`,
			expected: []netip.Prefix{
				netip.MustParsePrefix("100.24.149.244/32"),
				netip.MustParsePrefix("18.204.89.56/32"),
				netip.MustParsePrefix("3.89.170.186/32"),
			},
		},
		{
			name:     "Empty input",
			input:    `<html><body></body></html>`,
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

func TestIntegration_AmazonBot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	p := &AmazonParser{}
	data, err := fetchFromURL("https://developer.amazon.com/amazonbot/ip-addresses/")
	if err != nil {
		t.Skipf("network unavailable, skipping: %v", err)
	}

	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(result) == 0 {
		t.Error("AmazonBot IP list should not be empty")
	}

	// Verify all results are valid IPv4 prefixes
	for _, prefix := range result {
		if !prefix.Addr().Is4() {
			t.Errorf("expected IPv4 prefix, got %v", prefix)
		}
	}

	t.Logf("AmazonBot: parsed %d prefixes", len(result))
}
