package parser

import (
	"encoding/json"
	"io"
	"net/netip"
	"regexp"
)

type AmazonParser struct{}

func (p *AmazonParser) Name() string {
	return "amazon"
}

func (p *AmazonParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Extract JSON from HTML page - look for the code block with prefixes
	jsonPattern := regexp.MustCompile(`\{\s*"creationTime"\s*:\s*"[^"]+"\s*,\s*"prefixes"\s*:\s*\[([^\]]+)\]`)
	matches := jsonPattern.FindStringSubmatch(string(data))
	if len(matches) < 2 {
		// Try alternate pattern - the full JSON object
		jsonPattern2 := regexp.MustCompile(`\{[^{}]*"prefixes"\s*:\s*\[([^\]]*)\][^{}]*\}`)
		matches = jsonPattern2.FindStringSubmatch(string(data))
		if len(matches) < 2 {
			return nil, nil
		}
	}

	// Reconstruct valid JSON
	jsonStr := `{"prefixes": [` + matches[1] + `]}`

	var result struct {
		Prefixes []struct {
			IPV4Prefix string `json:"ipv4Prefix"`
		} `json:"prefixes"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// If simple pattern fails, try to parse individual IPs from the raw text
		return p.parseIPsFromText(string(data))
	}

	var prefixes []netip.Prefix
	for _, p := range result.Prefixes {
		if p.IPV4Prefix == "" {
			continue
		}
		// Try as CIDR first
		prefix, err := netip.ParsePrefix(p.IPV4Prefix)
		if err == nil {
			prefixes = append(prefixes, prefix)
			continue
		}
		// Try as individual IP
		addr, err := netip.ParseAddr(p.IPV4Prefix)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, 32))
	}
	return prefixes, nil
}

// parseIPsFromText extracts individual IPs from text and converts to /32 prefixes
func (p *AmazonParser) parseIPsFromText(data string) ([]netip.Prefix, error) {
	ipPattern := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	ips := ipPattern.FindAllString(data, -1)

	var prefixes []netip.Prefix
	seen := make(map[string]bool)
	for _, ip := range ips {
		if seen[ip] {
			continue
		}
		seen[ip] = true
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, 32))
	}
	return prefixes, nil
}

func init() {
	RegisterParser("amazon", &AmazonParser{})
}
