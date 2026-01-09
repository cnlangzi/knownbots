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
		WEBHOOKS []string `json:"WEBHOOKS"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse stripe json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, ip := range result.WEBHOOKS {
		if ip == "" {
			continue
		}
		addr, err := netip.ParseAddr(ip)
		if err == nil {
			prefix := netip.PrefixFrom(addr, addr.BitLen())
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes, nil
}

func init() {
	RegisterParser("stripe", &StripeParser{})
}
