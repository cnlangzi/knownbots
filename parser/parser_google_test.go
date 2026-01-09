package parser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func fetchFromURL(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

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

func TestGoogleParserWithRealFormat(t *testing.T) {
	p := &GoogleParser{}
	testInput := `{"prefixes": [{"ipv4Prefix": "66.249.64.0/19", "ipv6Prefix": "2001:db8::/32"}, {"ipv4Prefix": "142.250.0.0/15"}]}`
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

func TestIntegration_GoogleBot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("GoogleBot IP list should not be empty")
	}
	t.Logf("GoogleBot: parsed %d prefixes", len(result))
}

func TestIntegration_Bingbot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://www.bing.com/toolbox/bingbot.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("Bingbot IP list should not be empty")
	}
	t.Logf("Bingbot: parsed %d prefixes", len(result))
}

func TestIntegration_GPTBot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://openai.com/gptbot.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("GPTBot IP list should not be empty")
	}
	t.Logf("GPTBot: parsed %d prefixes", len(result))
}

func TestIntegration_Applebot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://search.developer.apple.com/applebot.json")
	if err != nil {
		t.Skipf("network unavailable, skipping: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("Applebot IP list should not be empty")
	}
	t.Logf("Applebot: parsed %d prefixes", len(result))
}
