package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type GitHubParser struct{}

func (p *GitHubParser) Name() string {
	return "github"
}

func (p *GitHubParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var result struct {
		Hooks []struct {
			IPV4Prefixes []struct {
				Prefix string `json:"prefix"`
			} `json:"ip_v4_prefixes"`
			IPV6Prefixes []struct {
				Prefix string `json:"prefix"`
			} `json:"ip_v6_prefixes"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse github json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, hook := range result.Hooks {
		for _, pfx := range hook.IPV4Prefixes {
			if pfx.Prefix != "" {
				prefix, err := netip.ParsePrefix(pfx.Prefix)
				if err == nil {
					prefixes = append(prefixes, prefix)
				}
			}
		}
		for _, pfx := range hook.IPV6Prefixes {
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
	RegisterParser("github", &GitHubParser{})
}
