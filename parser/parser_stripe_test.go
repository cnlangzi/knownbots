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
		{
			name:     "Mixed valid and invalid IPs",
			input:    `{"WEBHOOKS": ["3.18.12.63", "invalid-ip", "13.235.14.237", "not.an.ip"]}`,
			expected: 2,
		},
		{
			name:     "Empty strings in WEBHOOKS are ignored",
			input:    `{"WEBHOOKS": ["3.18.12.63", "", "13.235.14.237"]}`,
			expected: 2,
		},
		{
			name:     "IPv6 addresses are parsed",
			input:    `{"WEBHOOKS": ["3.18.12.63", "2001:db8::1"]}`,
			expected: 2,
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

func TestStripeParserErrorHandling(t *testing.T) {
	p := &StripeParser{}

	testCases := []struct {
		name        string
		input       string
		expectError bool
		expectedIPs int
	}{
		{
			name:        "Truncated JSON causes error",
			input:       `{"WEBHOOKS": ["3.18.`,
			expectError: true,
			expectedIPs: 0,
		},
		{
			name:        "Malformed JSON (missing brace)",
			input:       `{"WEBHOOKS": ["3.18.12.63"]`,
			expectError: true,
			expectedIPs: 0,
		},
		{
			name:        "Wrong type for WEBHOOKS (object instead of array)",
			input:       `{"WEBHOOKS": {"ip": "3.18.12.63"}}`,
			expectError: false,
			expectedIPs: 0,
		},
		{
			name:        "Wrong type for WEBHOOKS (string instead of array)",
			input:       `{"WEBHOOKS": "3.18.12.63"}`,
			expectError: false,
			expectedIPs: 0,
		},
		{
			name:        "Missing WEBHOOKS field",
			input:       `{"other": ["3.18.12.63"]}`,
			expectError: false,
			expectedIPs: 0,
		},
		{
			name:        "Non-JSON input causes error",
			input:       "not json at all",
			expectError: true,
			expectedIPs: 0,
		},
		{
			name:        "Numbers instead of strings in array",
			input:       `{"WEBHOOKS": [3.18, "13.235.14.237"]}`,
			expectError: false,
			expectedIPs: 1,
		},
		{
			name:        "Null in WEBHOOKS array",
			input:       `{"WEBHOOKS": [null, "13.235.14.237"]}`,
			expectError: false,
			expectedIPs: 1,
		},
		{
			name:        "All invalid IPs",
			input:       `{"WEBHOOKS": ["not-an-ip", "also-not-an-ip"]}`,
			expectError: false,
			expectedIPs: 0,
		},
		{
			name:        "Unicode in IP field",
			input:       `{"WEBHOOKS": ["3.18.12.63", "🦄"]}`,
			expectError: false,
			expectedIPs: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.Parse(strings.NewReader(tc.input))

			if tc.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(result) != tc.expectedIPs {
				t.Errorf("expected %d IPs, got %d", tc.expectedIPs, len(result))
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
