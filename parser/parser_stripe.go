package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type StripeParser struct{}

func (p *StripeParser) Name() string {
	return "stripe"
}

func (p *StripeParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var result struct {
		Webhooks []struct {
			IPV4Prefixes []struct {
				Prefix string `json:"ipv4_prefix"`
			} `json:"ipv4_prefixes"`
		} `json:"webhooks"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse stripe json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, wh := range result.Webhooks {
		for _, pfx := range wh.IPV4Prefixes {
			if pfx.Prefix != "" {
				prefix, err := netip.ParsePrefix(pfx.Prefix)
				if err == nil {
					prefixes = append(prefixes, prefix)
				}
			}
		}
	}
	return prefixes, nil
}

func init() {
	RegisterParser("stripe", &StripeParser{})
}
