package asn

import (
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"time"
)

// BGPHE fetches ASN prefixes from BGPHE (Hurricane Electric) website.
type BGPHE struct {
	Client *http.Client
}

// Match prefix links like: <a href="/net/31.13.24.0/21">31.13.24.0/21</a>
var bgpHePrefixRe = regexp.MustCompile(`<a href="/net/([0-9./a-fA-F]+)"[^>]*>`)

func NewBGPHE() *BGPHE {
	return &BGPHE{
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (b *BGPHE) Fetch(asn int) ([]netip.Prefix, error) {
	url := fmt.Sprintf("https://bgp.he.net/AS%d#_prefixes", asn)

	resp, err := b.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from BGPHE: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BGPHE returned status %d", resp.StatusCode)
	}

	// Read entire response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read BGPHE response: %w", err)
	}
	html := string(body)

	// Extract prefixes from HTML
	matches := bgpHePrefixRe.FindAllStringSubmatch(html, -1)

	prefixes := make([]netip.Prefix, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		prefixStr := strings.TrimSpace(match[1])
		if seen[prefixStr] {
			continue
		}
		seen[prefixStr] = true

		prefix, err := netip.ParsePrefix(prefixStr)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}
