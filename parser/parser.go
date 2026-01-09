package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Parser defines the interface for parsing IP ranges from remote sources.
type Parser interface {
	// Parse extracts CIDR strings from reader.
	Parse(r io.Reader) ([]string, error)
	// Name returns the parser identifier.
	Name() string
}

// parserRegistry holds all registered parsers.
var parserRegistry = make(map[string]Parser)

// RegisterParser registers a parser with the given name.
func RegisterParser(name string, p Parser) {
	parserRegistry[name] = p
}

// Get returns the parser for the given name, or falls back to txt parser.
func Get(name string) Parser {
	if p := parserRegistry[name]; p != nil {
		return p
	}
	return parserRegistry["txt"] // fallback to txt parser
}

// TxtParser parses line-by-line text format (IP/CIDR per line).
type TxtParser struct{}

func (p *TxtParser) Name() string {
	return "txt"
}

func (p *TxtParser) Parse(r io.Reader) ([]string, error) {
	var cidrs []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cidrs = append(cidrs, line)
	}
	return cidrs, scanner.Err()
}

// GoogleParser parses Google's JSON format: prefixes[].ipv4Prefix
type GoogleParser struct{}

func (p *GoogleParser) Name() string {
	return "google"
}

func (p *GoogleParser) Parse(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var result struct {
		Prefixes []struct {
			IPv4Prefix string `json:"ipv4Prefix"`
			IPv6Prefix string `json:"ipv6Prefix"`
		} `json:"prefixes"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse google json: %w", err)
	}

	var cidrs []string
	for _, prefix := range result.Prefixes {
		if prefix.IPv4Prefix != "" {
			cidrs = append(cidrs, prefix.IPv4Prefix)
		}
		if prefix.IPv6Prefix != "" {
			cidrs = append(cidrs, prefix.IPv6Prefix)
		}
	}
	return cidrs, nil
}

// OpenAIStyleParser parses OpenAI/Anthropic JSON format: prefixes[].prefix
type OpenAIStyleParser struct{}

func (p *OpenAIStyleParser) Name() string {
	return "openai"
}

func (p *OpenAIStyleParser) Parse(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var result struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse openai json: %w", err)
	}

	var cidrs []string
	for _, prefix := range result.Prefixes {
		if prefix.Prefix != "" {
			cidrs = append(cidrs, prefix.Prefix)
		}
	}
	return cidrs, nil
}

// GitHubParser parses GitHub API format: hooks[].ip_v4_prefixes[].prefix
type GitHubParser struct{}

func (p *GitHubParser) Name() string {
	return "github"
}

func (p *GitHubParser) Parse(r io.Reader) ([]string, error) {
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

	var cidrs []string
	for _, hook := range result.Hooks {
		for _, pfx := range hook.IPV4Prefixes {
			if pfx.Prefix != "" {
				cidrs = append(cidrs, pfx.Prefix)
			}
		}
		for _, pfx := range hook.IPV6Prefixes {
			if pfx.Prefix != "" {
				cidrs = append(cidrs, pfx.Prefix)
			}
		}
	}
	return cidrs, nil
}

// StripeParser parses Stripe format: webhooks[].ipv4_prefixes[].prefix
type StripeParser struct{}

func (p *StripeParser) Name() string {
	return "stripe"
}

func (p *StripeParser) Parse(r io.Reader) ([]string, error) {
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

	var cidrs []string
	for _, wh := range result.Webhooks {
		for _, pfx := range wh.IPV4Prefixes {
			if pfx.Prefix != "" {
				cidrs = append(cidrs, pfx.Prefix)
			}
		}
	}
	return cidrs, nil
}

func init() {
	// Register built-in parsers
	RegisterParser("txt", &TxtParser{})
	RegisterParser("google", &GoogleParser{})      // Google format
	RegisterParser("openai", &OpenAIStyleParser{}) // OpenAI/Anthropic format
	RegisterParser("github", &GitHubParser{})      // GitHub format
	RegisterParser("stripe", &StripeParser{})      // Stripe format
}
