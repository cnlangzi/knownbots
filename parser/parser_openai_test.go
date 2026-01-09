package parser

import (
	"net/netip"
	"strings"
	"testing"
)

func TestOpenAIStyleParser(t *testing.T) {
	p := &OpenAIStyleParser{}

	testCases := []struct {
		name     string
		input    string
		expected []netip.Prefix
	}{
		{
			name:  "OpenAI prefixes format",
			input: `{"prefixes": [{"prefix": "132.196.86.0/24"}, {"prefix": "172.182.202.0/25"}]}`,
			expected: []netip.Prefix{
				netip.MustParsePrefix("132.196.86.0/24"),
				netip.MustParsePrefix("172.182.202.0/25"),
			},
		},
		{
			name:     "Empty prefixes",
			input:    `{"prefixes": []}`,
			expected: []netip.Prefix{},
		},
		{
			name:  "Mixed empty and valid",
			input: `{"prefixes": [{"prefix": ""}, {"prefix": "1.2.3.0/24"}, {"prefix": ""}]}`,
			expected: []netip.Prefix{
				netip.MustParsePrefix("1.2.3.0/24"),
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

func TestOpenAIStyleParserWithRealFormat(t *testing.T) {
	p := &OpenAIStyleParser{}
	testInput := `{"prefixes": [{"prefix": "132.196.86.0/24"}, {"prefix": "172.182.202.0/25"}]}`
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(result))
	}
}

func TestOpenAIStyleParserInvalidPrefixes(t *testing.T) {
	p := &OpenAIStyleParser{}

	type testCase struct {
		name        string
		input       string
		expectError bool
		expected    int
	}

	testCases := []testCase{
		{
			name:        "Malformed CIDR (invalid mask)",
			input:       `{"prefixes": [{"prefix": "1.2.3.4/33"}]}`,
			expectError: false,
			expected:    0,
		},
		{
			name:        "Negative prefix length",
			input:       `{"prefixes": [{"prefix": "1.2.3.4/-1"}]}`,
			expectError: false,
			expected:    0,
		},
		{
			name:        "Invalid IP address in CIDR",
			input:       `{"prefixes": [{"prefix": "999.999.999.999/24"}]}`,
			expectError: false,
			expected:    0,
		},
		{
			name:        "Not a CIDR at all",
			input:       `{"prefixes": [{"prefix": "not-a-cidr"}]}`,
			expectError: false,
			expected:    0,
		},
		{
			name:        "Mixed valid and invalid prefixes",
			input:       `{"prefixes": [{"prefix": "1.2.3.0/24"}, {"prefix": "invalid"}, {"prefix": "5.6.7.0/24"}]}`,
			expectError: false,
			expected:    2,
		},
		{
			name:        "IPv6 with invalid prefix length",
			input:       `{"prefixes": [{"prefix": "2001:db8::/129"}]}`,
			expectError: false,
			expected:    0,
		},
		{
			name:        "Invalid IPv6 address",
			input:       `{"prefixes": [{"prefix": "gggg:gggg::/32"}]}`,
			expectError: false,
			expected:    0,
		},
		{
			name:        "Partial success with multiple invalid",
			input:       `{"prefixes": [{"prefix": "bad1"}, {"prefix": "1.2.3.0/24"}, {"prefix": "bad2"}, {"prefix": "5.6.7.0/24"}, {"prefix": "bad3"}]}`,
			expectError: false,
			expected:    2,
		},
		{
			name:        "Empty string prefix is ignored",
			input:       `{"prefixes": [{"prefix": ""}, {"prefix": "1.2.3.0/24"}]}`,
			expectError: false,
			expected:    1,
		},
		{
			name:        "Numbers instead of strings causes error",
			input:       `{"prefixes": [{"prefix": 123}]}`,
			expectError: true,
			expected:    0,
		},
		{
			name:        "Null in prefixes array",
			input:       `{"prefixes": [null]}`,
			expectError: false,
			expected:    0,
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
			if len(result) != tc.expected {
				t.Errorf("expected %d prefixes, got %d", tc.expected, len(result))
			}
		})
	}
}
