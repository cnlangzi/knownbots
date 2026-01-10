package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
)

type OpenAIParser struct{}

// openaiResponse represents the JSON response from OpenAI-style IP range APIs.
type openaiResponse struct {
	Prefixes []struct {
		Prefix string `json:"prefix"`
	} `json:"prefixes"`
}

func (p *OpenAIParser) Name() string {
	return "openai"
}

func (p *OpenAIParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var resp openaiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse openai json: %w", err)
	}

	var prefixes []netip.Prefix
	for _, pfx := range resp.Prefixes {
		if pfx.Prefix != "" {
			prefix, err := netip.ParsePrefix(pfx.Prefix)
			if err == nil {
				prefixes = append(prefixes, prefix)
			}
		}
	}
	return prefixes, nil
}

func init() {
	RegisterParser("openai", &OpenAIParser{})
}
