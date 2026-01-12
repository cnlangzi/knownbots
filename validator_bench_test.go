package knownbots

import (
	"os"
	"path/filepath"
	"testing"
)

func init() {
	EnableLog = false
}

// Benchmarks use -benchmem to track memory allocations:
//   - B/op: bytes allocated per operation (should be 0 for hot paths)
//   - allocs/op: number of allocations per operation (should be 0 for hot paths)
//   - Zero allocations = no GC pressure = stable QPS under load

// setupTestValidator creates a Validator with test bot configurations for benchmarking.
func setupTestValidator(tb testing.TB) *Validator {
	tmpDir := tb.TempDir()

	// Create conf.d subdirectory
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		tb.Fatalf("Failed to create conf.d: %v", err)
	}

	// Create multiple bot configurations to simulate real-world scenarios
	bots := []struct {
		name    string
		ua      string
		cidrs   []string
		domains []string
		urls    []string
	}{
		{
			name:    "googlebot",
			ua:      "Googlebot",
			cidrs:   []string{"66.249.64.0/19", "66.249.90.0/24", "66.249.91.0/24"},
			domains: []string{"googlebot.com"},
		},
		{
			name:    "bingbot",
			ua:      "Bingbot",
			cidrs:   []string{"40.76.0.0/14", "40.77.0.0/16", "40.78.0.0/15"},
			domains: []string{"bing.com", "msn.com"},
		},
		{
			name:    "slurp",
			ua:      "Slurp",
			cidrs:   []string{"199.96.0.0/12", "199.120.0.0/14"},
			domains: []string{"yahoo.com"},
		},
		{
			name:  "duckduckbot",
			ua:    "DuckDuckBot",
			cidrs: []string{"104.43.54.127/32", "20.50.48.192/32"},
			urls:  []string{"https://duckduckgo.com/duckduckbot.json"},
		},
		{
			name:    "yandexbot",
			ua:      "YandexBot",
			cidrs:   []string{"100.43.0.0/16", "95.108.0.0/16"},
			domains: []string{"yandex.com", "yandex.ru"},
		},
	}

	for _, bot := range bots {
		configContent := `name: ` + bot.name + `
ua: "` + bot.ua + `"
custom:
` + func() string {
			s := ""
			for _, cidr := range bot.cidrs {
				s += "  - \"" + cidr + "\"\n"
			}
			return s
		}() + func() string {
			if len(bot.domains) > 0 {
				s := "domains:\n"
				for _, domain := range bot.domains {
					s += "  - \"" + domain + "\"\n"
				}
				return s
			}
			return ""
		}() + func() string {
			if len(bot.urls) > 0 {
				s := "urls:\n"
				for _, url := range bot.urls {
					s += "  - \"" + url + "\"\n"
				}
				return s
			}
			return ""
		}()
		configPath := filepath.Join(confDir, bot.name+".yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			tb.Fatalf("Failed to write config for %s: %v", bot.name, err)
		}
	}

	v, err := New(WithRoot(tmpDir))
	if err != nil {
		tb.Fatalf("Failed to create validator: %v", err)
	}
	return v
}

// BenchmarkValidate_WithBotUA tests the full Validate path with matching bot UA and IP.
// This is the hottest path in production: legitimate bot verification.
func BenchmarkValidate_WithBotUA(b *testing.B) {
	v := setupTestValidator(b)
	defer v.Close()

	// Test cases: known bot UAs with verified IPs
	testCases := []struct {
		name string
		ua   string
		ip   string
	}{
		{"Googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "66.249.64.1"},
		{"Bingbot", "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", "40.76.0.1"},
		{"Slurp", "Mozilla/5.0 (compatible; Yahoo! Slurp; http://help.yahoo.com/help/us/ysearch/slurp)", "199.96.0.1"},
		{"DuckDuckBot", "Mozilla/5.0 (compatible; DuckDuckBot/1.0; +https://duckduckgo.com/duckduckbot)", "104.43.54.127"},
		{"YandexBot", "Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)", "100.43.0.1"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tc := testCases[i%len(testCases)]
			v.Validate(tc.ua, tc.ip)
			i++
		}
	})
}

// BenchmarkValidate_WithBotUA_IPMismatch tests bot UA with unverified IP.
// This path goes through RDNS verification.
func BenchmarkValidate_WithBotUA_IPMismatch(b *testing.B) {
	v := setupTestValidator(b)
	defer v.Close()

	testCases := []struct {
		name string
		ua   string
		ip   string
	}{
		{"Googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "1.2.3.4"},
		{"Bingbot", "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", "8.8.8.8"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tc := testCases[i%len(testCases)]
			v.Validate(tc.ua, tc.ip)
			i++
		}
	})
}

// BenchmarkValidate_BrowserUA tests legitimate browser UA classification.
// This is the second most common path in production.
func BenchmarkValidate_BrowserUA(b *testing.B) {
	v := setupTestValidator(b)
	defer v.Close()

	// Real browser User-Agents
	browserUAs := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ua := browserUAs[i%len(browserUAs)]
			v.Validate(ua, "192.168.1.1")
			i++
		}
	})
}

// BenchmarkValidate_UnknownBotUA tests unknown bot detection.
func BenchmarkValidate_UnknownBotUA(b *testing.B) {
	v := setupTestValidator(b)
	defer v.Close()

	unknownBotUAs := []string{
		"UnknownBot/1.0",
		"curl/7.68.0",
		"python-requests/2.28.0",
		"Bot/1.0",
		"Scrapy/2.9",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ua := unknownBotUAs[i%len(unknownBotUAs)]
			v.Validate(ua, "192.168.1.1")
			i++
		}
	})
}

// BenchmarkContainsIP tests IP range lookup performance.
// This is called for every IP verification.
func BenchmarkContainsIP(b *testing.B) {
	v := setupTestValidator(b)
	defer v.Close()

	testIPs := []string{
		"66.249.64.1",   // Googlebot
		"66.249.64.100", // Googlebot
		"40.76.0.1",     // Bingbot
		"40.77.0.1",     // Bingbot
		"1.2.3.4",       // Not in any range
		"199.96.0.1",    // Slurp
		"104.43.54.127", // DuckDuckBot
	}

	bot := v.getBots()[0] // Use Googlebot

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := testIPs[i%len(testIPs)]
			bot.ContainsIP(ip)
			i++
		}
	})
}

// BenchmarkFindBotByUA tests the UA matching performance.
// This is the first step in Validate() and uses lock-free reads.
func BenchmarkFindBotByUA(b *testing.B) {
	v := setupTestValidator(b)
	defer v.Close()

	testUAs := []string{
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"UnknownBot/1.0",
		"curl/7.68.0",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ua := testUAs[i%len(testUAs)]
			v.findBotByUA(ua)
			i++
		}
	})
}

// BenchmarkClassifyUA tests User-Agent classification performance.
// This is called for non-matching UAs.
func BenchmarkClassifyUA(b *testing.B) {
	testUAs := []string{
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"UnknownBot/1.0",
		"Mozilla/5.0",
		"curl/7.68.0",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ua := testUAs[i%len(testUAs)]
			classifyUA(ua)
			i++
		}
	})
}

// Benchmark_MixedTraffic simulates realistic production traffic distribution.
// Based on typical web traffic:
// - 60% legitimate browsers
// - 20% unknown bots/scrapers
// - 10% verified bots (search engines)
// - 10% unverified bots (suspicious)
func Benchmark_MixedTraffic(b *testing.B) {
	v := setupTestValidator(b)
	defer v.Close()

	// Traffic distribution
	browserUAs := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	}

	unknownBotUAs := []string{
		"UnknownBot/1.0",
		"curl/7.68.0",
		"python-requests/2.28.0",
		"Bot/1.0",
	}

	verifiedBotUAs := []struct {
		ua string
		ip string
	}{
		{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "66.249.64.1"},
		{"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", "40.76.0.1"},
	}

	unverifiedBotUAs := []struct {
		ua string
		ip string
	}{
		{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "1.2.3.4"},
		{"curl/7.68.0", "8.8.8.8"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 10 {
			case 0, 1, 2, 3, 4, 5: // 60% browsers
				ua := browserUAs[i%len(browserUAs)]
				v.Validate(ua, "192.168.1.1")
			case 6, 7: // 20% unknown bots
				ua := unknownBotUAs[i%len(unknownBotUAs)]
				v.Validate(ua, "192.168.1.1")
			case 8: // 10% verified bots
				tc := verifiedBotUAs[i%len(verifiedBotUAs)]
				v.Validate(tc.ua, tc.ip)
			case 9: // 10% unverified bots
				tc := unverifiedBotUAs[i%len(unverifiedBotUAs)]
				v.Validate(tc.ua, tc.ip)
			}
			i++
		}
	})
}
