package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type GoogleParser struct{}

type googleResponse struct {
	Prefixes []struct {
		IPv4Prefix string `json:"ipv4Prefix"`
		IPv6Prefix string `json:"ipv6Prefix"`
	} `json:"prefixes"`
}

func (p *GoogleParser) Name() string {
	return "google"
}

func (p *GoogleParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	var resp googleResponse
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to parse google json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, pfx := range resp.Prefixes {
		if pfx.IPv4Prefix != "" {
			prefix, err := netip.ParsePrefix(pfx.IPv4Prefix)
			if err == nil {
				prefixes = append(prefixes, prefix)
			}
		}
		if pfx.IPv6Prefix != "" {
			prefix, err := netip.ParsePrefix(pfx.IPv6Prefix)
			if err == nil {
				prefixes = append(prefixes, prefix)
			}
		}
	}
	return prefixes, nil
}

func init() {
	RegisterParser("google", &GoogleParser{})
}
