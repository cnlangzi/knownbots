package asn

import (
	"testing"
)

func TestIntegration_RIPE(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fetcher := NewRIPE()

	// Test with Google ASN (AS15169)
	prefixes, err := fetcher.Fetch(15169)
	if err != nil {
		t.Fatalf("failed to fetch prefixes from RIPE: %v", err)
	}

	if len(prefixes) == 0 {
		t.Error("expected some prefixes for AS15169, got none")
	}

	// Verify some expected Google prefixes are present
	foundGooglePrefix := false
	for _, p := range prefixes {
		if p.String() == "8.8.8.0/24" {
			foundGooglePrefix = true
			break
		}
	}
	if !foundGooglePrefix {
		t.Logf("did not find 8.8.8.0/24, but got %d prefixes total", len(prefixes))
	}

	// Verify all prefixes are valid
	for _, p := range prefixes {
		if !ValidatePrefix(p) {
			t.Errorf("invalid prefix: %s", p.String())
		}
	}

	t.Logf("RIPE fetched %d prefixes for AS15169 (Google)", len(prefixes))
}
