package asn

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"time"
)

// RouteViews fetches ASN prefixes from RouteViews API.
type RouteViews struct {
	Client *http.Client
}

func NewRouteViews() *RouteViews {
	return &RouteViews{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *RouteViews) Fetch(asn int) ([]netip.Prefix, error) {
	url := fmt.Sprintf("https://api.routeviews.org/asn/%d", asn)

	resp, err := r.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from RouteViews: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RouteViews API returned status %d: %s", resp.StatusCode, string(body))
	}

	// RouteViews returns a plain JSON array: ["8.8.8.0/24", ...]
	var prefixStrings []string
	if err := json.NewDecoder(resp.Body).Decode(&prefixStrings); err != nil {
		return nil, fmt.Errorf("failed to decode RouteViews response: %w", err)
	}

	prefixes := make([]netip.Prefix, 0, len(prefixStrings))
	for _, p := range prefixStrings {
		prefix, err := netip.ParsePrefix(p)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}
