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
