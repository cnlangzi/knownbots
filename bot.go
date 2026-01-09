package knownbots

import (
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

// IPPrefix is a type alias for net/netip.Prefix for better performance.
type IPPrefix = netip.Prefix

// Bot represents the configuration for a single bot.
type Bot struct {
	Name    string        `yaml:"name"`
	UA      string        `yaml:"ua"`
	URLs    []string      `yaml:"urls"`
	custom  *atomic.Value // []IPPrefix, atomic for lock-free reads
	Domains []string      `yaml:"domains"`
	RDNS    bool          `yaml:"rdns"` // whether to perform RDNS verification
	Cache   *Cache        // RDNS cache, initialized by Validator
	fail    *LRU          // failed IP cache for fast rejection
}

// Load loads all bot configurations from the bots directory.
// It reads all .yaml or .yml files from the conf.d subdirectory.
func Load(dir string) ([]*Bot, error) {
	confDir := filepath.Join(dir, "conf.d")
	entries, err := os.ReadDir(confDir)
	if err != nil {
		return nil, err
	}

	var bots []*Bot
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(confDir, entry.Name())
		bot, err := loadBot(path)
		if err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}

	return bots, nil
}

// loadBot loads a single bot configuration file.
func loadBot(path string) (*Bot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tmp struct {
		Name    string   `yaml:"name"`
		UA      string   `yaml:"ua"`
		URLs    []string `yaml:"urls"`
		Custom  []string `yaml:"custom"`
		Domains []string `yaml:"domains"`
		RDNS    bool     `yaml:"rdns"`
	}
	if err := yaml.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}

	customNets := parseCIDRs(tmp.Custom)
	customValue := &atomic.Value{}
	customValue.Store(customNets)

	return &Bot{
		Name:    tmp.Name,
		UA:      tmp.UA,
		URLs:    tmp.URLs,
		custom:  customValue,
		Domains: tmp.Domains,
		RDNS:    tmp.RDNS,
	}, nil
}

// parseCIDRs converts CIDR strings to IPPrefix.
func parseCIDRs(cidrs []string) []IPPrefix {
	var nets []IPPrefix
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil {
			nets = append(nets, prefix)
		}
	}
	return nets
}

// SetCustom atomically updates the custom IP list.
func (b *Bot) SetCustom(cidrs []string) {
	nets := parseCIDRs(cidrs)
	b.custom.Store(nets)
}

// ContainsIP checks if the IP is in the bot's custom IP ranges.
func (b *Bot) ContainsIP(ipStr string) bool {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}

	custom := b.custom.Load().([]IPPrefix)
	for _, prefix := range custom {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// VerifyRDNS checks if the IP's reverse DNS hostname matches this bot's domains.
// It uses the bot's cache for performance.
func (b *Bot) VerifyRDNS(ipStr string) bool {
	cache := b.Cache

	// Check fail cache first (fast rejection)
	if b.fail.Contains(ipStr) {
		return false
	}

	// Check valid cache first
	if hostname, ok := cache.Get(ipStr); ok {
		return matchDomain(hostname, b.Domains)
	}

	// Perform RDNS lookup
	names, err := net.LookupAddr(ipStr)
	if err != nil || len(names) == 0 {
		// Network error or no records - mark as failed
		b.fail.Add(ipStr)
		return false
	}

	hostname := strings.TrimSuffix(names[0], ".")
	if matchDomain(hostname, b.Domains) {
		cache.Set(ipStr, hostname)
		return true
	}

	// Valid RDNS but not matching domain - mark as failed
	b.fail.Add(ipStr)
	return false
}
