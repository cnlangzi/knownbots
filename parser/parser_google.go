package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type GoogleParser struct{}

func (p *GoogleParser) Name() string {
	return "google"
}

func (p *GoogleParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var result struct {
		Prefixes []struct {
			IPv4Prefix string `json:"ipv4Prefix"`
			IPv6Prefix string `json:"ipv6Prefix"`
		} `json:"prefixes"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse google json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, pfx := range result.Prefixes {
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
