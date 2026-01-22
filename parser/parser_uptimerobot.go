package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type UptimeRobotParser struct{}

type uptimeRobotResponse struct {
	Prefixes []struct {
		IPPrefix   string `json:"ip_prefix"`
		IPv6Prefix string `json:"ipv6_prefix"`
	} `json:"prefixes"`
}

func (p *UptimeRobotParser) Name() string {
	return "uptimerobot"
}

func (p *UptimeRobotParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	var resp uptimeRobotResponse
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to parse uptimerobot json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, pfx := range resp.Prefixes {
		if pfx.IPPrefix != "" {
			prefix, err := netip.ParsePrefix(pfx.IPPrefix)
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
	RegisterParser("uptimerobot", &UptimeRobotParser{})
}
