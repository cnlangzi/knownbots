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
	if len(matches) >= 2 {
		// Reconstruct valid JSON
		jsonStr := `{"prefixes": [` + matches[1] + `]}`

		var result struct {
			Prefixes []struct {
				IPV4Prefix string `json:"ipv4Prefix"`
			} `json:"prefixes"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil && len(result.Prefixes) > 0 {
			return p.extractPrefixes(result.Prefixes), nil
		}
	}

	// Try alternate pattern - the full JSON object
	jsonPattern2 := regexp.MustCompile(`\{[^{}]*"prefixes"\s*:\s*\[([^\]]*)\][^{}]*\}`)
	matches = jsonPattern2.FindStringSubmatch(string(data))
	if len(matches) >= 2 {
		jsonStr := `{"prefixes": [` + matches[1] + `]}`
		var result struct {
			Prefixes []struct {
				IPV4Prefix string `json:"ipv4Prefix"`
			} `json:"prefixes"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil && len(result.Prefixes) > 0 {
			return p.extractPrefixes(result.Prefixes), nil
		}
	}

	// Fallback: parse individual IPs from text
	return p.parseIPsFromText(string(data))
}

func (p *AmazonParser) extractPrefixes(prefixes []struct {
	IPV4Prefix string `json:"ipv4Prefix"`
}) []netip.Prefix {
	var result []netip.Prefix
	for _, pref := range prefixes {
		if pref.IPV4Prefix == "" {
			continue
		}
		// Try as CIDR first
		prefix, err := netip.ParsePrefix(pref.IPV4Prefix)
		if err == nil {
			result = append(result, prefix)
			continue
		}
		// Try as individual IP
		addr, err := netip.ParseAddr(pref.IPV4Prefix)
		if err != nil {
			continue
		}
		result = append(result, netip.PrefixFrom(addr, 32))
	}
	return result
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
