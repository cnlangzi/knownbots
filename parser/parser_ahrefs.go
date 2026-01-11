package parser

import (
	"encoding/json"
	"io"
	"net/netip"
)

type AhrefsParser struct{}

func (p *AhrefsParser) Name() string {
	return "ahrefs"
}

func (p *AhrefsParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	var data struct {
		IPs []struct {
			IPAddress string `json:"ip_address"`
		} `json:"ips"`
	}
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}

	var prefixes []netip.Prefix
	for _, ip := range data.IPs {
		addr, err := netip.ParseAddr(ip.IPAddress)
		if err != nil {
			continue
		}
		bits := 32
		if addr.Is6() {
			bits = 128
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, bits))
	}
	return prefixes, nil
}

func init() {
	RegisterParser("ahrefs", &AhrefsParser{})
}
