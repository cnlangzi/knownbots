package knownbots

import (
	"log"
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

// BotKind represents the category of a bot based on intent/behavior.
// See classification matrix in bots/conf.d/ for full reference.
type BotKind string

// botConfig represents the YAML configuration for a bot.
type botConfig struct {
	Name    string   `yaml:"name"`
	Kind    BotKind  `yaml:"kind"`
	Parser  string   `yaml:"parser"`
	UA      string   `yaml:"ua"`
	URLs    []string `yaml:"urls"`
	Custom  []string `yaml:"custom"`
	Domains []string `yaml:"domains"`
	RDNS    bool     `yaml:"rdns"`
}

const (
	KindSearchEngine BotKind = "SearchEngine" // Search engine crawlers (Googlebot, Bingbot)
	KindSocialMedia  BotKind = "SocialMedia"  // Social media preview fetchers (FacebookBot, Twitterbot)
	KindAITraining   BotKind = "AITraining"   // AI model trainers - only take, no return (GPTBot, ClaudeBot)
	KindAIAssist     BotKind = "AIAssist"     // AI assistants - search & answer, may bring traffic (PerplexityBot)
	KindAIMixed      BotKind = "AIMixed"      // Mixed AI purposes (Google-Extended)
	KindSEO          BotKind = "SEO"          // SEO backlink analyzers (AhrefsBot, SemrushBot)
	KindMonitor      BotKind = "Monitor"      // Uptime/ad verification (Pingdom, AdsBot)
	KindSecurity     BotKind = "Security"     // Security scanners (Censys, Shodan)
	KindScraper      BotKind = "Scraper"      // Content/price scrapers, HTTP client libraries and CLI tools
	KindUnknown      BotKind = "Unknown"      // Unclassified
)

// Bot represents the configuration for a single bot.
type Bot struct {
	Name    string                      `yaml:"name"`
	Kind    BotKind                     `yaml:"kind"`
	Parser  string                      `yaml:"parser"` // parser name, defaults to bot name
	UA      string                      `yaml:"ua"`
	URLs    []string                    `yaml:"urls"`
	custom  *atomic.Pointer[[]IPPrefix] // []IPPrefix, atomic for lock-free reads
	Domains []string                    `yaml:"domains"`
	RDNS    bool                        `yaml:"rdns"` // whether to perform RDNS verification
	Cache   *Cache                      // RDNS cache, initialized by Validator
	fail    *LRU                        // failed IP cache for fast rejection
}

// Load loads all bot configurations from the bots directory.
// It loads built-in bots from embedded configuration and then
// loads custom bots from the conf.d subdirectory.
// Custom bots override built-in bots with the same name.
func Load(dir string) ([]*Bot, error) {
	// First, load all built-in bots
	embedded, err := loadEmbedded()
	if err != nil {
		return nil, err
	}

	// Then, load custom bots from user's directory
	customDir := filepath.Join(dir, "conf.d")
	entries, err := os.ReadDir(customDir)
	if err != nil {
		// Only treat "dir doesn't exist" as "use built-in only"
		// Return other errors (permission, I/O) so callers can handle them
		if os.IsNotExist(err) {
			bots := make([]*Bot, 0, len(embedded))
			for _, bot := range embedded {
				bots = append(bots, bot)
			}
			return bots, nil
		}
		return nil, err
	}

	// Merge: start with embedded, then override with custom
	bots := embedded

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(customDir, entry.Name())
		bot, err := loadBot(path)
		if err != nil {
			return nil, err
		}
		if bot == nil {
			continue
		}

		// Custom bot overrides built-in bot with the same name
		if _, exists := embedded[bot.Name]; exists {
			if EnableLog {
				log.Printf("[knownbots] custom config %q overrides built-in bot", bot.Name)
			}
		}
		bots[bot.Name] = bot
	}

	// Convert map to slice
	result := make([]*Bot, 0, len(bots))
	for _, bot := range bots {
		result = append(result, bot)
	}

	return result, nil
}

// loadBot loads a single bot configuration file.
func loadBot(path string) (*Bot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg botConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Validate required Name field
	if cfg.Name == "" {
		if EnableLog {
			log.Printf("[knownbots] skip %q: missing required 'name' field", path)
		}
		return nil, nil
	}

	// Use bot name as default parser if not specified
	parser := cfg.Parser
	if parser == "" {
		parser = cfg.Name
	}

	customNets := parseCIDRs(cfg.Custom)
	customValue := &atomic.Pointer[[]IPPrefix]{}
	customValue.Store(&customNets)

	return &Bot{
		Name:    cfg.Name,
		Kind:    cfg.Kind,
		Parser:  parser,
		UA:      cfg.UA,
		URLs:    cfg.URLs,
		custom:  customValue,
		Domains: cfg.Domains,
		RDNS:    cfg.RDNS,
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

// ContainsIP checks if the IP is in the bot's custom IP ranges.
func (b *Bot) ContainsIP(ipStr string) bool {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}

	custom := b.custom.Load()
	for _, prefix := range *custom {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// VerifyRDNS checks if the IP's reverse DNS hostname matches this bot's domains.
// It uses the bot's cache for performance.
func (b *Bot) VerifyRDNS(ipStr string) ResultStatus {
	cache := b.Cache

	// Check fail cache first (fast rejection)
	if b.fail.Contains(ipStr) {
		return StatusFailed
	}

	// Check valid cache first
	if hostname, ok := cache.Get(ipStr); ok {
		if matchDomain(hostname, b.Domains) {
			return StatusVerified
		}
		return StatusFailed
	}

	// Perform RDNS lookup
	names, err := net.LookupAddr(ipStr)
	if err != nil {
		// Distinguish network errors from NXDOMAIN (no PTR record)
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			// NXDOMAIN: PTR record does not exist - verification failed
			b.fail.Add(ipStr)
			return StatusFailed
		}
		// Network error (timeout, connection issue, etc.) - allow retry
		return StatusPending
	}
	if len(names) == 0 {
		// No PTR records - verification failed, add to fail cache
		b.fail.Add(ipStr)
		return StatusFailed
	}

	hostname := strings.TrimSuffix(names[0], ".")
	if matchDomain(hostname, b.Domains) {
		cache.Set(ipStr, hostname)
		return StatusVerified
	}

	// Valid RDNS but not matching domain - mark as failed
	b.fail.Add(ipStr)
	return StatusFailed
}
