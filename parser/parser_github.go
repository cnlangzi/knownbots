package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type GitHubParser struct{}

// githubIPResponse represents the JSON response from GitHub's IP API.
type githubIPResponse struct {
	Hooks []string `json:"hooks"`
	Web   []string `json:"web"`
	API   []string `json:"api"`
}

func (p *GitHubParser) Name() string {
	return "github"
}

func (p *GitHubParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var resp githubIPResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse github json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, cidr := range resp.Hooks {
		if cidr != "" {
			prefix, err := netip.ParsePrefix(cidr)
			if err == nil {
				prefixes = append(prefixes, prefix)
			}
		}
	}
	for _, cidr := range resp.Web {
		if cidr != "" {
			prefix, err := netip.ParsePrefix(cidr)
			if err == nil {
				prefixes = append(prefixes, prefix)
			}
		}
	}
	for _, cidr := range resp.API {
		if cidr != "" {
			prefix, err := netip.ParsePrefix(cidr)
			if err == nil {
				prefixes = append(prefixes, prefix)
			}
		}
	}
	return prefixes, nil
}

func init() {
	RegisterParser("github", &GitHubParser{})
}
