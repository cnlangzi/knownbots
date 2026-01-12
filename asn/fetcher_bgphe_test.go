package asn

import (
	"testing"
)

func TestIntegration_BGPHE(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fetcher := NewBGPHE()

	// Test with Facebook ASN (AS32934) - has many prefixes
	prefixes, err := fetcher.Fetch(32934)
	if err != nil {
		t.Fatalf("failed to fetch prefixes from BGPHE: %v", err)
	}

	if len(prefixes) == 0 {
		t.Error("expected some prefixes for AS32934, got none")
	}

	// Verify all prefixes are valid
	for _, p := range prefixes {
		if !ValidatePrefix(p) {
			t.Errorf("invalid prefix: %s", p.String())
		}
	}

	t.Logf("BGPHE fetched %d prefixes for AS32934 (Facebook)", len(prefixes))
}
