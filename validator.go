package knownbots

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/cnlangzi/knownbots/parser"
)

// Default settings
const (
	FailLRULimit      = 1000
	SchedulerInterval = 24 * time.Hour
)

// ResultStatus represents the verification result status.
type ResultStatus string

const (
	StatusVerified ResultStatus = "verified"
	StatusFailed   ResultStatus = "failed"
	StatusUnknown  ResultStatus = "unknown"
)

// Result represents the verification result.
type Result struct {
	Name       string       `json:"name"`
	Status     ResultStatus `json:"status"`
	IsBot      bool         `json:"is_bot"`
	IsVerified bool         `json:"is_verified"`
}

// Validator is the core bot verification engine.
type Validator struct {
	root       string
	bots       atomic.Value // []*Bot, atomic for lock-free reads
	uaIndex    atomic.Value // map[byte][]*Bot, byte-level index for UA lookup
	cancel     context.CancelFunc
	interval   time.Duration
	failLimit  int
	classifyUA bool
}

// getBots returns the current bots slice atomically.
func (v *Validator) getBots() []*Bot {
	return v.bots.Load().([]*Bot)
}

// setBots stores the bots slice atomically and builds the UA index.
func (v *Validator) setBots(bots []*Bot) {
	v.bots.Store(bots)
	v.uaIndex.Store(buildUAIndex(bots))
}

// buildUAIndex creates a byte-level index mapping first characters to bot candidates.
func buildUAIndex(bots []*Bot) map[byte][]*Bot {
	index := make(map[byte][]*Bot, 26)

	for _, bot := range bots {
		if bot.UA == "" {
			continue
		}

		firstChar := bot.UA[0]
		index[firstChar] = append(index[firstChar], bot)
	}

	return index
}

// Config holds the options for creating a Validator.
type Config struct {
	Root       string
	Interval   time.Duration
	FailLimit  int
	ClassifyUA bool
}

// Option is a functional option for configuring a Validator.
type Option func(*Config)

// WithRoot sets the bots root directory (containing conf.d and data subdirs).
func WithRoot(dir string) Option {
	return func(c *Config) {
		c.Root = dir
	}
}

// WithSchedulerInterval sets the background scheduler interval.
func WithSchedulerInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.Interval = interval
	}
}

// WithFailLimit sets the limit for failed lookup cache.
func WithFailLimit(limit int) Option {
	return func(c *Config) {
		c.FailLimit = limit
	}
}

// WithClassifyUA enables UA classification for non-bot UAs.
// By default, classifyUA is disabled for performance.
// Enable it to distinguish legitimate browsers from suspicious UAs.
func WithClassifyUA() Option {
	return func(c *Config) {
		c.ClassifyUA = true
	}
}

// New creates a new Validator instance with background scheduler.
func New(opts ...Option) (*Validator, error) {
	cfg := Config{
		Root:       "./bots",
		Interval:   SchedulerInterval,
		FailLimit:  FailLRULimit,
		ClassifyUA: false, // Default: skip classifyUA for performance
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	bots, err := Load(cfg.Root)
	if err != nil {
		return nil, err
	}

	for _, bot := range bots {
		if !bot.RDNS {
			continue
		}
		cache, err := NewCache(filepath.Join(cfg.Root, bot.Name, "rdns.txt"))
		if err != nil {
			return nil, err
		}
		bot.Cache = cache
		bot.fail = NewLRU(cfg.FailLimit)
	}

	ctx, cancel := context.WithCancel(context.Background())
	v := &Validator{
		root:       cfg.Root,
		cancel:     cancel,
		interval:   cfg.Interval,
		failLimit:  cfg.FailLimit,
		classifyUA: cfg.ClassifyUA,
	}
	v.setBots(bots)

	go v.startScheduler(ctx)

	return v, nil
}

// startScheduler runs background tasks:
//   - refreshIPs: download and update IP ranges from official URLs
//   - pruneCaches: verify and clean up cached RDNS entries
//   - persistCaches: write valid cache entries to persistent storage
func (v *Validator) startScheduler(ctx context.Context) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	ticker := time.NewTicker(v.interval)
	defer ticker.Stop()

	// Run immediately on start
	v.runScheduler(httpClient)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			v.runScheduler(httpClient)
		}
	}
}

func (v *Validator) runScheduler(httpClient *http.Client) {
	bots := v.getBots()

	// Refresh IP ranges: update memory first, then persist
	for _, bot := range bots {
		newIPs := downloadIPs(httpClient, bot)
		if len(newIPs) == 0 {
			continue
		}

		// Update memory (effective immediately)
		bot.SetCustom(newIPs)

		// Persist to file (optional, failure is OK)
		path := filepath.Join(v.root, bot.Name, "ips.txt")
		writeIPs(path, newIPs)
	}

	for _, bot := range bots {
		if !bot.RDNS || bot.Cache == nil {
			continue
		}

		cache := bot.Cache
		cache.Prune(bot.Domains)

		if err := cache.Persist(); err != nil {
			log.Printf("[knownbots] failed to persist cache for %s: %v", bot.Name, err)
		}
	}
}

// downloadIPs fetches IP ranges from URLs and parses using the bot's registered parser.
func downloadIPs(httpClient *http.Client, bot *Bot) []netip.Prefix {
	var allPrefixes []netip.Prefix
	p := parser.Get(bot.Parser)

	for _, url := range bot.URLs {
		resp, err := httpClient.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		prefixes, err := p.Parse(resp.Body)
		if err != nil {
			log.Printf("[knownbots] failed to parse IPs from %s: %v", url, err)
			continue
		}
		allPrefixes = append(allPrefixes, prefixes...)
	}
	return allPrefixes
}

// writeIPs persists IP ranges to file (failure is OK).
func writeIPs(path string, prefixes []netip.Prefix) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, prefix := range prefixes {
		if _, err := fmt.Fprintln(w, prefix.String()); err != nil {
			return
		}
	}
	w.Flush()
}

// Validate verifies if the given UserAgent and IP belong to a known bot.
// Returns a Result with:
//   - IsBot: true if UA matches a known bot, false if it's a legitimate browser
//   - IsVerified: true if the IP is verified for the bot
//   - Status: verified (bot confirmed), failed (bot suspected, IP not verified), or unknown (not a bot or browser)
func (v *Validator) Validate(ua, ip string) Result {
	// Step 1: Check if UA matches any known bot (claims to be a known bot)
	if bot := v.findBotByUA(ua); bot != nil {
		result := v.verifyIP(bot, ip)
		result.IsBot = true
		return result
	}

	// Step 2: Classify UA type (single pass)
	if v.classifyUA {
		switch classifyUA(ua) {
		case Browser:
			// Valid browser structure → not a bot
			return Result{Status: StatusUnknown, IsBot: false, IsVerified: false}
		case Suspicious:
			// Claims to be browser but malformed → suspicious bot
			return Result{Status: StatusUnknown, IsBot: true, IsVerified: false}
		default:
			//unknown bot
			return Result{Status: StatusUnknown, IsBot: true, IsVerified: false}
		}
	}

	// classifyUA disabled: unknown UA, assume not a bot
	return Result{Status: StatusUnknown, IsBot: false, IsVerified: false}
}

// verifyIP verifies if the IP belongs to the given bot.
func (v *Validator) verifyIP(bot *Bot, ipStr string) Result {
	if bot.ContainsIP(ipStr) {
		return Result{Name: bot.Name, Status: StatusVerified, IsVerified: true}
	}

	if bot.RDNS && bot.VerifyRDNS(ipStr) {
		return Result{Name: bot.Name, Status: StatusVerified, IsVerified: true}
	}

	return Result{Name: bot.Name, Status: StatusFailed, IsVerified: false}
}

// findBotByUA finds a bot by matching the UserAgent marker.
// Uses byte-level index for fast lookup, then validates with word boundary matching.
func (v *Validator) findBotByUA(ua string) *Bot {
	if len(ua) == 0 {
		return nil
	}

	index := v.uaIndex.Load().(map[byte][]*Bot)

	for i := 0; i < len(ua); i++ {
		candidates := index[ua[i]]
		if len(candidates) == 0 {
			continue
		}

		for _, bot := range candidates {
			if containsWord(ua, bot.UA) {
				return bot
			}
		}
	}

	return nil
}

// Reload reloads all bot configurations.
func (v *Validator) Reload(root string) error {
	bots, err := Load(root)
	if err != nil {
		return err
	}

	for _, bot := range bots {
		if !bot.RDNS {
			continue
		}
		if bot.Cache == nil {
			cache, err := NewCache(filepath.Join(root, bot.Name, "rdns.txt"))
			if err != nil {
				return err
			}
			bot.Cache = cache
		}
		if bot.fail == nil {
			bot.fail = NewLRU(v.failLimit)
		}
	}

	v.root = root
	v.setBots(bots)
	return nil
}

// Close stops the scheduler.
func (v *Validator) Close() error {
	v.cancel()
	return nil
}
