package knownbots

import (
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create conf.d subdirectory
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("Failed to create conf.d: %v", err)
	}

	// Write test config
	configContent := `name: testbot
ua: "TestBot"
urls:
  - "https://example.com/ips.json"
custom:
  - "192.168.1.0/24"
  - "10.0.0.0/8"
domains:
  - "example.com"
rdns: false
`
	configPath := filepath.Join(confDir, "testbot.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load bots (includes built-in bots)
	bots, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load bots: %v", err)
	}

	// Should have all built-in bots + 1 custom = 58
	expectedCount := EmbeddedBotCount() + 1
	if len(bots) != expectedCount {
		t.Fatalf("Expected %d bots (built-in + custom), got %d", expectedCount, len(bots))
	}

	// Find the custom bot
	var found bool
	for _, bot := range bots {
		if bot.Name == "testbot" {
			found = true
			if bot.UA != "TestBot" {
				t.Errorf("Expected UA 'TestBot', got '%s'", bot.UA)
			}
		}
	}
	if !found {
		t.Error("Expected to find custom testbot")
	}
}

func TestLoadWithOverride(t *testing.T) {
	// Test that custom config overrides built-in config
	tmpDir := t.TempDir()

	// Create conf.d subdirectory
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("Failed to create conf.d: %v", err)
	}

	// Override built-in googlebot with custom config
	configContent := `name: googlebot
ua: "CustomGooglebot"
custom:
  - "1.2.3.0/24"
domains: []
rdns: false
`
	configPath := filepath.Join(confDir, "googlebot.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load bots
	bots, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load bots: %v", err)
	}

	// Should have same count as built-in (custom replaces built-in, not adds)
	expectedCount := EmbeddedBotCount()
	if len(bots) != expectedCount {
		t.Errorf("Expected %d bots (built-in count), got %d", expectedCount, len(bots))
	}

	// Find googlebot and verify it's the custom one
	var found bool
	for _, bot := range bots {
		if bot.Name == "googlebot" {
			found = true
			if bot.UA != "CustomGooglebot" {
				t.Errorf("Expected UA 'CustomGooglebot' (override), got '%s'", bot.UA)
			}
			if len(bot.Domains) != 0 {
				t.Errorf("Expected empty domains (override), got %v", bot.Domains)
			}
		}
	}
	if !found {
		t.Error("Expected to find googlebot")
	}
}

func TestValidator(t *testing.T) {
	tmpDir := t.TempDir()

	// Create conf.d subdirectory
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("Failed to create conf.d: %v", err)
	}

	// Create config
	configContent := `name: googlebot
ua: "Googlebot"
custom:
  - "66.249.64.0/19"
domains:
  - "googlebot.com"
urls: []
rdns: false
`
	configPath := filepath.Join(confDir, "googlebot.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	v, err := New(WithRoot(tmpDir), WithClassifyUA())
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	defer v.Close()

	// Test UA matching
	ua := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	result := v.Validate(ua, "66.249.64.1")
	if result.Status != StatusVerified {
		t.Errorf("Expected verified, got %s", result.Status)
	}

	// Test non-matching UA - malformed browser-like UA should be marked as bot
	result = v.Validate("Mozilla/5.0", "66.249.64.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %s", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for malformed browser-like UA")
	}
}

func TestContainsIP(t *testing.T) {
	tests := []struct {
		cidr  string
		ip    string
		match bool
	}{
		{"192.168.1.0/24", "192.168.1.100", true},
		{"192.168.1.0/24", "192.168.2.1", false},
		{"10.0.0.0/8", "10.255.255.255", true},
		{"10.0.0.0/8", "11.0.0.1", false},
	}

	for _, tt := range tests {
		bot := &Bot{
			custom: &atomic.Value{},
		}
		prefix, _ := netip.ParsePrefix(tt.cidr)
		bot.SetCustom([]netip.Prefix{prefix})
		result := bot.ContainsIP(tt.ip)
		if result != tt.match {
			t.Errorf("ContainsIP(%s, %s) = %v, want %v", tt.cidr, tt.ip, result, tt.match)
		}
	}
}

func TestClassifyUA(t *testing.T) {
	tests := []struct {
		ua     string
		expect BrowserKind
	}{
		// Legitimate browsers
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", Browser},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1", Browser},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0", Browser},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15", Browser},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0", Browser},
		{"Opera/9.80 (Windows NT 6.1; WOW64) Presto/2.12.388 Version/12.18", Browser},
		// Suspicious (malformed browser-like)
		{"Mozilla/5.0", Suspicious},
		{"Mozilla/5.0 AppleWebKit/537.36", Suspicious},
		{"Mozilla/5.0 Gecko/20100101", Suspicious},
		{"AppleWebKit/537.36 Chrome/120.0.0.0", Suspicious},
		{"Mozilla", Suspicious},
		{"Mozilla/5.0 (X11)", Suspicious},
		{"Mozilla/5.0 (Windows NT 10.0)", Suspicious},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 14_0) AppleWebKit/605.1.15 Gecko/20100101", Suspicious},
		{"Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 (iPhone; CPU iPhone OS 14_0)", Suspicious},
		{"Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) Gecko/20100101 Firefox/95.0", Suspicious},
		{"Mozilla/5.0 (compatible; Bot; \\(paren\\))", Suspicious},
		{"Mozilla/5.0 (unbalanced \\(", Suspicious},
		{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", Unknown},
		{"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", Unknown},
		{"Mozilla/5.0 (compatible; DuckDuckBot/1.0; +https://duckduckgo.com/duckduckbot)", Unknown},
		{"Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)", Unknown},
		{"", Unknown},
		{"Googlebot/2.1", Unknown},
		{"curl/7.68.0", Unknown},
		{"python-requests/2.28.0", Unknown},
		{"Bot/1.0", Unknown},
		{"UnknownBot/1.0", Unknown},
	}

	for _, tt := range tests {
		result := classifyUA(tt.ua)
		if result != tt.expect {
			t.Errorf("classifyUA(%q) = %v, want %v", tt.ua, result, tt.expect)
		}
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		text   string
		word   string
		expect bool
	}{
		// Exact match
		{"Googlebot/2.1", "Googlebot", true},
		// At start
		{"Googlebot is here", "Googlebot", true},
		// At end
		{"Hello Googlebot", "Googlebot", true},
		// In middle
		{"Hello Googlebot test", "Googlebot", true},
		// Not a word (prefix)
		{"SuperGooglebot", "Googlebot", false},
		// Not a word (suffix)
		{"GooglebotPro", "Googlebot", false},
		// Empty word
		{"Some text", "", false},
		// Empty text
		{"", "Googlebot", false},
		// Multiple occurrences
		{"Googlebot and Googlebot", "Googlebot", true},
		// With special characters as boundaries
		{"(Googlebot)", "Googlebot", true},
		{"[Googlebot]", "Googlebot", true},
		{"-Googlebot-", "Googlebot", true},
	}

	for _, tt := range tests {
		result := containsWord(tt.text, tt.word)
		if result != tt.expect {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, result, tt.expect)
		}
	}
}

func TestValidatorWithBrowserDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create conf.d subdirectory
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("Failed to create conf.d: %v", err)
	}

	// Create config
	configContent := `name: googlebot
ua: "Googlebot"
custom:
  - "66.249.64.0/19"
domains:
  - "googlebot.com"
urls: []
rdns: false
`
	configPath := filepath.Join(confDir, "googlebot.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	v, err := New(WithRoot(tmpDir), WithClassifyUA())
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	defer v.Close()

	// Test 1: Known bot with verified IP
	ua := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	result := v.Validate(ua, "66.249.64.1")
	if result.Status != StatusVerified {
		t.Errorf("Expected verified, got %s", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for known bot")
	}
	if !result.IsVerified {
		t.Error("Expected IsVerified=true for verified IP")
	}

	// Test 2: Known bot with unverified IP
	result = v.Validate(ua, "1.2.3.4")
	if result.Status != StatusFailed {
		t.Errorf("Expected failed, got %s", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for known bot")
	}
	if result.IsVerified {
		t.Error("Expected IsVerified=false for unverified IP")
	}

	// Test 3: Legitimate browser (not a bot)
	browserUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	result = v.Validate(browserUA, "192.168.1.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %s", result.Status)
	}
	if result.IsBot {
		t.Error("Expected IsBot=false for legitimate browser")
	}

	// Test 4: Unknown bot (not a known bot, not a browser)
	result = v.Validate("UnknownBot/1.0", "192.168.1.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %s", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for unknown bot")
	}

	// Test 5: Malformed browser-like UA (likely a bot pretending to be a browser)
	malformedUA := "Mozilla/5.0 AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	result = v.Validate(malformedUA, "192.168.1.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %s", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for malformed browser-like UA")
	}
}

func TestValidatorDefaultBehavior(t *testing.T) {
	// Test default behavior (classifyUA disabled for performance)
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("Failed to create conf.d: %v", err)
	}

	configContent := `name: googlebot
ua: "Googlebot"
custom:
  - "66.249.64.0/19"
domains:
  - "googlebot.com"
urls: []
rdns: false
`
	configPath := filepath.Join(confDir, "googlebot.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Default: classifyUA disabled
	v, err := New(WithRoot(tmpDir))
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	defer v.Close()

	// Unknown UA (not a known bot) should return IsBot=false when classifyUA is disabled
	result := v.Validate("UnknownBot/1.0", "192.168.1.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %s", result.Status)
	}
	if result.IsBot {
		t.Error("Expected IsBot=false for unknown UA when classifyUA is disabled")
	}
}
