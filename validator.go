package knownbots

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"
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
	bots       atomic.Pointer[[]*Bot]          // []*Bot, atomic for lock-free reads
	uaIndex    atomic.Pointer[map[byte][]*Bot] // map[byte][]*Bot, byte-level index for UA lookup
	cancel     context.CancelFunc
	interval   time.Duration
	failLimit  int
	classifyUA bool
}

// getBots returns the current bots slice atomically.
func (v *Validator) getBots() []*Bot {
	return *v.bots.Load()
}

// setBots stores the bots slice atomically and builds the UA index.
func (v *Validator) setBots(bots []*Bot) {
	uaIndex := buildUAIndex(bots)
	v.bots.Store(&bots)
	v.uaIndex.Store(&uaIndex)
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
			if EnableLog {
				log.Printf("[knownbots] failed to persist cache for %s: %v", bot.Name, err)
			}
		}
	}
}

// Validate verifies if the given UserAgent and IP belong to a known bot.
// By default (classifyUA disabled), unknown UAs return IsBot=false for performance.
// When WithClassifyUA() is enabled:
//   - IsBot: true if UA matches a known bot or is suspicious, false if it's a legitimate browser
//   - IsVerified: true if the IP is verified for the bot
//   - Status: verified (bot confirmed), failed (bot suspected, IP not verified), or unknown
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
			// Unknown (not browser-like)
			return Result{Status: StatusUnknown, IsBot: true, IsVerified: false}
		}
	}

	// classifyUA disabled (default): unknown UA, assume not a bot
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

	index := v.uaIndex.Load()

	for i := 0; i < len(ua); i++ {
		candidates := (*index)[ua[i]]
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
