package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type StripeParser struct{}

// stripeIPResponse represents the JSON response from Stripe's webhook IP API.
type stripeIPResponse struct {
	WEBHOOKS []string `json:"WEBHOOKS"`
}

func (p *StripeParser) Name() string {
	return "stripe"
}

func (p *StripeParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var resp stripeIPResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse stripe json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, ip := range resp.WEBHOOKS {
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
