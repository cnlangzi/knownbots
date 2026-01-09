package parser

import (
	"strings"
	"testing"
)

func TestStripeParser(t *testing.T) {
	p := &StripeParser{}

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "Stripe WEBHOOKS format",
			input:    `{"WEBHOOKS": ["3.18.12.63", "3.130.192.231", "13.235.14.237"]}`,
			expected: 3,
		},
		{
			name:     "Empty WEBHOOKS",
			input:    `{"WEBHOOKS": []}`,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tc.expected {
				t.Errorf("expected %d prefixes, got %d", tc.expected, len(result))
			}
		})
	}
}

func TestStripeParserWithRealFormat(t *testing.T) {
	p := &StripeParser{}
	testInput := `{"WEBHOOKS": ["3.18.12.63", "3.130.192.231", "13.235.14.237"]}`
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

func TestIntegration_Stripe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &StripeParser{}
	data, err := fetchFromURL("https://stripe.com/files/ips/ips_webhooks.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("Stripe webhook IP list should not be empty")
	}
	t.Logf("Stripe: parsed %d IPs", len(result))
}
