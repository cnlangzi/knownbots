# knownbots

[![Go Reference](https://pkg.go.dev/badge/github.com/cnlangzi/knownbots.svg)](https://pkg.go.dev/github.com/cnlangzi/knownbots)
[![Go Report Card](https://goreportcard.com/badge/github.com/cnlangzi/knownbots)](https://goreportcard.com/report/github.com/cnlangzi/knownbots)

**knownbots** is a high-performance Go library for verifying search engine crawlers and identifying legitimate bots. It protects your web services from bot impersonation by validating User-Agent strings and IP addresses through RDNS lookups and IP range verification.

## Why knownbots?

**The Problem**: Malicious actors can easily spoof User-Agent strings to impersonate legitimate search engine bots (Googlebot, Bingbot, etc.) to bypass rate limits, scrape content, or exploit bot-specific logic.

**The Solution**: knownbots performs **cryptographic-strength verification** by:
1. **Matching User-Agent markers** (case-sensitive word boundaries)
2. **Verifying IP ownership** through reverse DNS lookups or official IP ranges
3. **Caching results** to avoid expensive DNS queries on subsequent requests

## Key Features

### 🚀 High Performance
- **Lock-free reads** via `atomic.Value` for bot configuration and RDNS cache
- **Zero-allocation hot paths** using `netip.Prefix` for IP matching
- **Byte-level indexing** for O(1) bot lookup (150-300ns for 40 bots vs 640ns linear scan)
- **Copy-on-Write caching** optimized for read-heavy workloads (1-20 writes/day)

### 🔒 Security First
- **Case-sensitive matching** prevents forgery attempts (official bots use fixed casing)
- **Word boundary validation** prevents partial matches (e.g., "MyGooglebot" won't match)
- **LRU fail cache** for fast rejection of known-bad IPs (1000 entry limit)
- **Browser detection** distinguishes legitimate users from suspicious bot-like patterns

### 📦 Production Ready
- **Persistent RDNS cache** survives restarts (file-based storage)
- **Background scheduler** automatically refreshes IP ranges from official URLs
- **Graceful degradation** (cache persistence failures don't affect runtime)
- **Comprehensive tests** with benchmarks for 3-40 bot scenarios

### 🌍 Extensible
- **YAML-based configuration** for easy bot additions (no code changes)
- **Pluggable verification** supports both IP ranges and RDNS verification
- **Official source integration** automatically downloads and updates IP lists

## Installation

```bash
go get github.com/cnlangzi/knownbots
```

**Requirements**: Go 1.21+

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/cnlangzi/knownbots"
)

func main() {
    // Initialize validator (starts background scheduler)
    v, err := knownbots.New()
    if err != nil {
        log.Fatal(err)
    }
    defer v.Close()
    
    // Verify a bot claim
    result := v.Validate(
        "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
        "66.249.66.1",
    )
    
    fmt.Printf("Status: %s\n", result.Status)      // "verified"
    fmt.Printf("IsBot: %t\n", result.IsBot)        // true
    fmt.Printf("IsVerified: %t\n", result.IsVerified) // true
    fmt.Printf("Bot Name: %s\n", result.Name)      // "googlebot"
}
```

### HTTP Middleware Example

```go
func BotVerificationMiddleware(v *knownbots.Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ua := r.Header.Get("User-Agent")
            ip := r.RemoteAddr // In production, extract from X-Forwarded-For
            
            result := v.Validate(ua, ip)
            
            // Block fake bots (claims to be bot but IP not verified)
            if result.IsBot && !result.IsVerified {
                http.Error(w, "Forbidden: Bot verification failed", http.StatusForbidden)
                return
            }
            
            // Add verification metadata to request context
            ctx := context.WithValue(r.Context(), "botVerified", result)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Configuration Options

```go
v, err := knownbots.New(
    knownbots.WithRoot("./custom-bots"),           // Custom bot config directory
    knownbots.WithSchedulerInterval(12*time.Hour), // IP refresh frequency
    knownbots.WithFailLimit(5000),                 // Failed lookup cache size
)
```

## Configuration

### Directory Structure

```
bots/
├── conf.d/              # Bot configurations (YAML)
│   ├── googlebot.yaml
│   ├── bingbot.yaml
│   └── ...
├── googlebot/           # Bot-specific data (auto-created)
│   ├── rdns.txt        # Persistent RDNS cache
│   └── ips.txt         # Downloaded IP ranges
└── ...
```

### Bot Configuration (YAML)

```yaml
name: googlebot
ua: "Googlebot"                           # EXACT casing required (case-sensitive)
urls:                                      # Official IP list URLs (auto-downloaded)
  - "https://www.gstatic.com/ipranges/google.json"
custom:                                    # Static CIDR ranges (always checked)
  - "66.249.64.0/19"
domains:                                   # Verified RDNS domains
  - "googlebot.com"
  - "google.com"
rdns: true                             # Enable RDNS verification (false = IP-only)
```

**Important**: 
- User-Agent markers (`ua`) are **case-sensitive**. Official bots use fixed casing (e.g., "Googlebot", never "googlebot"). This prevents forgery attempts where attackers alter casing to bypass detection.
- Set `rdns: false` for bots that only need IP range verification (faster, no DNS queries)

### Adding New Bots

1. Create `bots/conf.d/newbot.yaml` with bot configuration
2. Reload validator: `v.Reload("./bots")` (or restart application)
3. No code changes required!

## How It Works

### Verification Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     Incoming Request                         │
│                  (User-Agent + IP Address)                   │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
         ┌────────────────────┐
         │  UA Matches Bot?   │──No──▶ Classify UA Type
         └────────┬───────────┘        (Browser/Suspicious/Unknown)
                  │ Yes                         │
                  ▼                             ▼
         ┌────────────────────┐        Return: IsBot=false
         │  Check IP Ranges   │        (legitimate browser)
         │  (CIDR matching)   │
         └────────┬───────────┘
                  │
                  ├─ Hit ──▶ Return: verified
                  │
                  ├─ Miss + rdns=false ──▶ Return: failed
                  │
                  ▼
         ┌────────────────────┐
         │   Bot.RDNS=true?   │──No──▶ Return: failed
         └────────┬───────────┘        (IP-only bot, no DNS check)
                  │ Yes
                  ▼
         ┌────────────────────┐
         │  Check Fail Cache  │──Hit──▶ Return: failed
         │  (LRU, 1000 IPs)   │        (known fake bot)
         └────────┬───────────┘
                  │ Miss
                  ▼
         ┌────────────────────┐
         │ Check RDNS Cache   │──Hit──▶ Domain match?
         │  (persistent)      │         Yes: verified
         └────────┬───────────┘         No: failed
                  │ Miss
                  ▼
         ┌────────────────────┐
         │ Perform RDNS Lookup│──▶ Domain match?
         │  (50-200ms delay)  │     Yes: verified + cache
         └────────────────────┘     No: failed + fail cache
```

### Background Scheduler (Every 24h)

```
┌─────────────────────────────────────────────────────────────┐
│                    Background Scheduler                      │
└─────────────────┬───────────────────────────────────────────┘
                  │
        ┌─────────┴─────────┐
        │                   │
        ▼                   ▼
  ┌──────────┐      ┌──────────────┐
  │ Refresh  │      │ Prune & Save │
  │ IP Lists │      │ RDNS Cache   │
  │ (HTTP)   │      │ (rdns=true)  │
  └──────────┘      └──────────────┘
        │                   │
        ▼                   ▼
  Update memory      Remove invalid
  Persist to file    Persist to file
                     (only for rdns=true bots)
```

## Performance

### Benchmarks (40 bots, Intel i5-1038NG7 @ 2.00GHz)

| Operation | Time/op | Allocs/op | Notes |
|-----------|---------|-----------|-------|
| **UA matching (hit first)** | 165ns | 0 | Byte index + word boundary check |
| **UA matching (hit middle)** | 300ns | 0 | Worst case: mid-list match |
| **UA matching (miss)** | 640ns | 0 | Full scan + browser classification |
| **Validate (cache hit)** | 227ns | 0 | IP range check only |
| **Validate (RDNS hit)** | 450ns | 0 | Cache lookup + domain match |
| **Validate (cold lookup)** | 50-200ms | 1-2 | DNS query (first time only) |

**Key Insight**: After initial RDNS lookup, all subsequent verifications for the same IP are **sub-microsecond** (227-450ns).

### Scalability

| Bot Count | Index Benefit | Recommended Index |
|-----------|---------------|-------------------|
| < 20 bots | Minimal (2x) | Single byte (current) |
| 20-50 bots | Significant (4-5x) | Single byte (current) |
| > 50 bots | Critical (10x+) | Consider 3-char prefix |

Current implementation is optimized for **3-50 bots** (covers 99% of use cases).

## API Reference

### Types

```go
type Validator struct { /* ... */ }

type Result struct {
    Name       string       // Bot name (e.g., "googlebot")
    Status     ResultStatus // "verified" | "failed" | "unknown"
    IsBot      bool         // True if UA matches any bot or looks bot-like
    IsVerified bool         // True if IP ownership verified
}

type ResultStatus string
const (
    StatusVerified ResultStatus = "verified" // Bot confirmed (UA + IP match)
    StatusFailed   ResultStatus = "failed"   // Bot suspected but IP invalid
    StatusUnknown  ResultStatus = "unknown"  // Not a known bot
)
```

### Methods

```go
// New creates a validator with background scheduler
func New(opts ...Option) (*Validator, error)

// Validate verifies User-Agent and IP address
func (v *Validator) Validate(ua, ip string) Result

// Reload reloads bot configurations from disk
func (v *Validator) Reload(root string) error

// Close stops background scheduler
func (v *Validator) Close() error
```

### Options

```go
// WithRoot sets custom bot directory (default: "./bots")
func WithRoot(dir string) Option

// WithSchedulerInterval sets refresh interval (default: 24h)
func WithSchedulerInterval(interval time.Duration) Option

// WithFailLimit sets failed lookup cache size (default: 1000)
func WithFailLimit(limit int) Option
```

## Real-World Use Cases

### 1. Rate Limiting
```go
// Apply different rate limits for verified bots vs browsers
result := validator.Validate(ua, ip)
if result.IsVerified {
    limiter = rateLimits.Bot  // Generous: 10/sec
} else if result.IsBot {
    limiter = rateLimits.FakeBot  // Strict: 1/min
} else {
    limiter = rateLimits.Browser  // Normal: 5/sec
}
```

### 2. Analytics Exclusion
```go
// Exclude verified bots from user analytics
result := validator.Validate(ua, ip)
if !result.IsBot || !result.IsVerified {
    analytics.Track(userID, event)
}
```

### 3. SEO Testing
```go
// Allow verified Googlebot to bypass feature flags
result := validator.Validate(ua, ip)
if result.Name == "googlebot" && result.IsVerified {
    features.EnableAll()  // Show production content for indexing
}
```

### 4. Content Protection
```go
// Block fake bots from scraping paywalled content
result := validator.Validate(ua, ip)
if result.IsBot && !result.IsVerified {
    return http.StatusForbidden  // Suspected scraper
}
```

## Supported Bots (Built-in Configs)

Current built-in configurations:
- **Googlebot** (Google Search)
- **Bingbot** (Microsoft Bing)
- **facebookexternalhit** (Facebook/Meta link previews)

**Need more bots?** Add YAML configs to `bots/conf.d/` - no code changes required!

**Common bots to add**:
- Yandex (YandexBot)
- Baidu (Baiduspider)
- DuckDuckGo (DuckDuckBot)
- Twitter (Twitterbot)
- Slack (Slackbot)
- OpenAI (GPTBot)
- Anthropic (anthropic-ai)
- Apple (Applebot)

See [`bots/conf.d/googlebot.yaml`](bots/conf.d/googlebot.yaml) for configuration examples.

## Testing

```bash
# Run all tests
go test ./...

# Run benchmarks
go test -bench=. -benchmem

# Run specific test
go test -v -run ^TestValidator$

# Coverage report
go test -cover ./...
```

## Architecture Decisions

### Why atomic.Value instead of RWMutex?
Bot configurations change rarely (on reload/schedule, 1-20x/day) but are read on every request (1000s/sec). Lock-free reads via `atomic.Value` eliminate contention and provide consistent sub-microsecond performance.

### Why case-sensitive UA matching?
Official bots use **fixed casing** ("Googlebot", never "googlebot"). Case variations indicate forgery. Case-sensitive matching:
1. Rejects fakes at first stage (no expensive DNS queries)
2. 4x faster than case-insensitive (16ns vs 67ns)
3. Improves both security and performance

### Why Copy-on-Write cache?
RDNS cache sees 1-20 new IPs per day but 1000s of reads per second (99.99% read ratio). Copy-on-Write with atomic swap provides:
- Zero-allocation reads (no locking)
- Safe concurrent access
- Simple implementation (vs lock-free data structures)

### Why byte-level index?
Linear bot list scan is fast for 3 bots (52ns) but degrades to 640ns at 40 bots. Single-character index provides 4-5x speedup for 20-50 bots at minimal memory cost (<1KB).

## Contributing

Contributions are welcome! Please:

1. Add bot configurations to `bots/conf.d/` (ensure `ua` matches official casing)
2. Write tests for new functionality
3. Run benchmarks to verify no performance regressions
4. Follow [standard Go conventions](https://go.dev/doc/effective_go)

## License

[MIT License](LICENSE)

## Author

**cnlangzi** - [GitHub](https://github.com/cnlangzi)

## Acknowledgments

- Inspired by Google's official [bot verification documentation](https://developers.google.com/search/docs/crawling-indexing/verifying-googlebot)
- Performance patterns influenced by Go stdlib's `sync/atomic` and `net/netip` designs
- Special thanks to all contributors and users providing feedback

---

**⭐ Star this project if you find it useful!**

**📝 Questions?** Open an issue or start a discussion!

**🐛 Found a bug?** Please report it with minimal reproduction steps!
