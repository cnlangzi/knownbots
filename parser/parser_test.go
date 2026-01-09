package parser

import (
	"strings"
	"testing"
)

func TestGoogleParser(t *testing.T) {
	parser := &GoogleParser{}

	testCases := []struct {
		name     string
		input    string
		expected []string
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
			expected: []string{"66.249.64.0/19", "2001:db8::/32", "66.249.90.0/24", "2600:1900::/28"},
		},
		{
			name: "Special crawlers format",
			input: `{
				"prefixes": [
					{"ipv4Prefix": "66.249.64.0/19"},
					{"ipv4Prefix": "142.250.0.0/15"}
				]
			}`,
			expected: []string{"66.249.64.0/19", "142.250.0.0/15"},
		},
		{
			name:     "Empty prefixes",
			input:    `{"prefixes": []}`,
			expected: []string{},
		},
		{
			name: "Only IPv4",
			input: `{
				"prefixes": [
					{"ipv4Prefix": "66.249.64.0/19"},
					{"ipv4Prefix": "66.249.90.0/24"}
				]
			}`,
			expected: []string{"66.249.64.0/19", "66.249.90.0/24"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parser.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d prefixes, got %d", len(tc.expected), len(result))
			}

			for i, exp := range tc.expected {
				if result[i] != exp {
					t.Errorf("expected prefix %d to be %q, got %q", i, exp, result[i])
				}
			}
		})
	}
}

func TestTxtParser(t *testing.T) {
	parser := &TxtParser{}

	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple line-by-line",
			input:    "1.2.3.4/24\n5.6.7.8/24\n9.10.11.12",
			expected: []string{"1.2.3.4/24", "5.6.7.8/24", "9.10.11.12"},
		},
		{
			name:     "With comments and empty lines",
			input:    "# Comment line\n1.2.3.4/24\n\n5.6.7.8/24\n# Another comment",
			expected: []string{"1.2.3.4/24", "5.6.7.8/24"},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Only comments",
			input:    "# Just a comment\n# Another comment",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parser.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d lines, got %d", len(tc.expected), len(result))
			}

			for i, exp := range tc.expected {
				if result[i] != exp {
					t.Errorf("expected line %d to be %q, got %q", i, exp, result[i])
				}
			}
		})
	}
}

func TestParserRegistry(t *testing.T) {
	// Test that parsers are registered
	parsers := []string{"txt", "google", "openai", "github", "stripe"}
	for _, name := range parsers {
		t.Run(name, func(t *testing.T) {
			p := Get(name)
			if p == nil {
				t.Errorf("parser %q not found in registry", name)
			}
			if p.Name() != name {
				t.Errorf("parser name mismatch: expected %q, got %q", name, p.Name())
			}
		})
	}

	// Test fallback to txt parser
	t.Run("fallback to txt", func(t *testing.T) {
		p := Get("nonexistent")
		if p == nil {
			t.Fatal("fallback parser should not be nil")
		}
		if p.Name() != "txt" {
			t.Errorf("fallback parser should be txt, got %q", p.Name())
		}
	})
}
