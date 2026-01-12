package knownbots

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/cnlangzi/knownbots/asn"
)

// Default settings
const (
	FailLRULimit      = 1000
	SchedulerInterval = 24 * time.Hour
)

// ResultStatus represents the verification result status.
type ResultStatus int

const (
	StatusVerified ResultStatus = 1 // IP verified successfully
	StatusPending  ResultStatus = 2 // RDNS network error, can retry
	StatusFailed   ResultStatus = 3 // IP not matched, suspected fake bot
	StatusUnknown  ResultStatus = 0 // Not a bot (normal browser)
)

// Result represents the verification result.
type Result struct {
	BotName string       `json:"bot_name"`
	BotKind BotKind      `json:"bot_kind"`
	IsBot   bool         `json:"is_bot"`
	Status  ResultStatus `json:"status"`
}

// Validator is the core bot verification engine.
type Validator struct {
	root       string
	bots       atomic.Pointer[[]*Bot]          // []*Bot, atomic for lock-free reads
	uaIndex    atomic.Pointer[map[byte][]*Bot] // map[byte][]*Bot, byte-level index for UA lookup
	asnStore   *asn.Store
	cancel     context.CancelFunc
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

	// Initialize ASN store first
	asnStore, err := asn.NewStore(cfg.Root)
	if err != nil {
		return nil, err
	}

	for _, bot := range bots {
		botDir := filepath.Join(cfg.Root, bot.Name)
		if err := os.MkdirAll(botDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}

		bot.initIPs(filepath.Join(botDir, "ips.txt"))
		bot.initASN(asnStore)
		bot.initRDNS(filepath.Join(botDir, "rdns.txt"))
		bot.fail = NewLRU(cfg.FailLimit)
	}

	ctx, cancel := context.WithCancel(context.Background())
	v := &Validator{
		root:       cfg.Root,
		asnStore:   asnStore,
		cancel:     cancel,
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
	ticker := time.NewTicker(SchedulerInterval)
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

	for _, bot := range bots {
		// Update IP ranges
		bot.refreshIPs(httpClient, v.root)
		// Update ASN data  with ASN configured
		bot.refreshASN(v.asnStore)
		// Prune and persist RDNS caches
		bot.refreshRDNS()
	}
}

// Validate verifies if the given UserAgent and IP belong to a known bot.
// By default (classifyUA disabled), unknown UAs return IsBot=false for performance.
// When WithClassifyUA() enabled:
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
			return Result{Status: StatusUnknown, BotKind: KindUnknown, IsBot: false}
		case Suspicious:
			// Claims to be browser but malformed → suspicious bot
			return Result{Status: StatusUnknown, BotKind: KindUnknown, IsBot: true}
		default:
			// Unknown (not browser-like)
			return Result{Status: StatusUnknown, BotKind: KindUnknown, IsBot: true}
		}
	}

	// classifyUA disabled (default): unknown UA, assume not a bot
	return Result{Status: StatusUnknown, BotKind: KindUnknown, IsBot: false}
}

// verifyIP verifies if the IP belongs to the given bot.
// Verification order: IP ranges → ASN → RDNS (fastest to slowest)
func (v *Validator) verifyIP(bot *Bot, ipStr string) Result {
	// Check IP ranges first (fastest, ~200ns)
	if bot.ContainsIP(ipStr) {
		return Result{BotName: bot.Name, BotKind: bot.Kind, Status: StatusVerified, IsBot: true}
	}

	// ASN verification (fast after cache load, ~100ns)
	if bot.asns != nil && bot.asns.Contains(netip.MustParseAddr(ipStr)) {
		return Result{BotName: bot.Name, BotKind: bot.Kind, Status: StatusVerified, IsBot: true}
	}

	// RDNS verification (50-200ms cold, ~450ns cached)
	if bot.RDNS {
		switch bot.VerifyRDNS(ipStr) {
		case StatusVerified:
			return Result{BotName: bot.Name, BotKind: bot.Kind, Status: StatusVerified, IsBot: true}
		case StatusPending:
			return Result{BotName: bot.Name, BotKind: bot.Kind, Status: StatusPending, IsBot: true}
		default:
			return Result{BotName: bot.Name, BotKind: bot.Kind, Status: StatusFailed, IsBot: true}
		}
	}

	// No match found
	return Result{BotName: bot.Name, BotKind: bot.Kind, Status: StatusFailed, IsBot: true}
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

// Close stops the scheduler.
func (v *Validator) Close() error {
	v.cancel()
	return nil
}
