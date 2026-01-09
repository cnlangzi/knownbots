package parser

import (
	"strings"
	"testing"
)

func TestGitHubParser(t *testing.T) {
	p := &GitHubParser{}

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "GitHub hooks, web, api",
			input:    `{"hooks": ["192.30.252.0/22"], "web": ["192.30.252.0/22"], "api": ["192.30.252.0/22"]}`,
			expected: 3,
		},
		{
			name:     "Empty arrays",
			input:    `{"hooks": [], "web": [], "api": []}`,
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

func TestGitHubParserWithRealFormat(t *testing.T) {
	p := &GitHubParser{}
	// Test input with known structure: 3 CIDRs in each category (hooks, web, api)
	testInput := `{"hooks": ["192.30.252.0/22", "192.30.253.0/24", "192.30.254.0/24"], "web": ["192.30.252.0/22", "192.30.253.0/24", "192.30.254.0/24"], "api": ["192.30.252.0/22", "192.30.253.0/24", "192.30.254.0/24"]}`
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	// Expected: 3 categories × 3 CIDRs each = 9 total
	if len(result) != 9 {
		t.Errorf("expected 9 prefixes (3 hooks + 3 web + 3 api), got %d", len(result))
	}
	// Verify the CIDRs are correctly parsed
	expectedPrefixes := []string{
		"192.30.252.0/22", "192.30.253.0/24", "192.30.254.0/24", // hooks
		"192.30.252.0/22", "192.30.253.0/24", "192.30.254.0/24", // web
		"192.30.252.0/22", "192.30.253.0/24", "192.30.254.0/24", // api
	}
	for i, exp := range expectedPrefixes {
		if result[i].String() != exp {
			t.Errorf("expected prefix %d to be %s, got %s", i, exp, result[i].String())
		}
	}
}

func TestIntegration_GitHub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GitHubParser{}
	data, err := fetchFromURL("https://api.github.com/meta")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("GitHub IP list should not be empty")
	}
	t.Logf("GitHub: parsed %d prefixes", len(result))
}
