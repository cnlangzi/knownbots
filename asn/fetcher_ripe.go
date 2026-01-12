package asn

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"time"
)

// RIPE fetches ASN prefixes from RIPE Stat API.
type RIPE struct {
	Client *http.Client
}

type RIPEData struct {
	Data struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	} `json:"data"`
}

func NewRIPE() *RIPE {
	return &RIPE{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *RIPE) Fetch(asn int) ([]netip.Prefix, error) {
	url := fmt.Sprintf("https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS%d", asn)

	resp, err := r.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from RIPE: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RIPE API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data RIPEData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode RIPE response: %w", err)
	}

	prefixes := make([]netip.Prefix, 0, len(data.Data.Prefixes))
	for _, p := range data.Data.Prefixes {
		prefix, err := netip.ParsePrefix(p.Prefix)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}
