package knownbots

import (
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func init() {
	EnableLog = false
}

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

	// Should have at least our custom bot
	if len(bots) == 0 {
		t.Fatal("Expected at least one bot")
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

	// Should have at least some bots (built-in + custom)
	if len(bots) == 0 {
		t.Fatal("Expected at least one bot")
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
		t.Errorf("Expected verified, got %d", result.Status)
	}

	// Test non-matching UA - malformed browser-like UA should be marked as bot
	result = v.Validate("Mozilla/5.0", "66.249.64.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %d", result.Status)
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
			custom: &atomic.Pointer[[]IPPrefix]{},
		}
		prefix, _ := netip.ParsePrefix(tt.cidr)
		bot.custom.Store(&[]netip.Prefix{prefix})
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
		t.Errorf("Expected verified, got %d", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for known bot")
	}
	if result.Status != StatusVerified {
		t.Error("Expected Status=Verified for verified IP")
	}

	// Test 2: Known bot with unverified IP
	result = v.Validate(ua, "1.2.3.4")
	if result.Status != StatusFailed {
		t.Errorf("Expected failed, got %d", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for known bot")
	}
	if result.Status == StatusVerified {
		t.Error("Expected Status!=Verified for unverified IP")
	}

	// Test 3: Legitimate browser (not a bot)
	browserUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	result = v.Validate(browserUA, "192.168.1.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %d", result.Status)
	}
	if result.IsBot {
		t.Error("Expected IsBot=false for legitimate browser")
	}

	// Test 4: Unknown bot (not a known bot, not a browser)
	result = v.Validate("UnknownBot/1.0", "192.168.1.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %d", result.Status)
	}
	if !result.IsBot {
		t.Error("Expected IsBot=true for unknown bot")
	}

	// Test 5: Malformed browser-like UA (likely a bot pretending to be a browser)
	malformedUA := "Mozilla/5.0 AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	result = v.Validate(malformedUA, "192.168.1.1")
	if result.Status != StatusUnknown {
		t.Errorf("Expected unknown, got %d", result.Status)
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
		t.Errorf("Expected unknown, got %d", result.Status)
	}
	if result.IsBot {
		t.Error("Expected IsBot=false for unknown UA when classifyUA is disabled")
	}
}

func TestMatchDomain(t *testing.T) {
	tests := []struct {
		hostname string
		domains  []string
		expect   bool
	}{
		{"google.com", []string{"google.com"}, true},
		{"www.google.com", []string{"google.com"}, true},
		{"mail.google.com", []string{"google.com"}, true},
		{"google.com", []string{"example.com"}, false},
		{"notgoogle.com", []string{"google.com"}, false},
		{"", []string{"google.com"}, false},
		{"google.com", []string{}, false},
	}

	for _, tt := range tests {
		result := matchDomain(tt.hostname, tt.domains)
		if result != tt.expect {
			t.Errorf("matchDomain(%q, %v) = %v, want %v", tt.hostname, tt.domains, result, tt.expect)
		}
	}
}

func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		c      byte
		expect bool
	}{
		{'a', true}, {'z', true}, {'A', true}, {'Z', true}, {'0', true}, {'9', true},
		{' ', false}, {'.', false}, {'-', false}, {'@', false},
	}

	for _, tt := range tests {
		result := isAlphaNumeric(tt.c)
		if result != tt.expect {
			t.Errorf("isAlphaNumeric(%q) = %v, want %v", tt.c, result, tt.expect)
		}
	}
}

func TestSplitTwo(t *testing.T) {
	tests := []struct {
		input   string
		expect1 string
		expect2 string
	}{
		{"key value", "key", "value"},
		{"single", "", ""},
		{"", "", ""},
		{"no space here", "no", "space here"},
	}

	for _, tt := range tests {
		a, b := splitTwo(tt.input)
		if a != tt.expect1 || b != tt.expect2 {
			t.Errorf("splitTwo(%q) = (%q, %q), want (%q, %q)", tt.input, a, b, tt.expect1, tt.expect2)
		}
	}
}

func TestBuildUAIndex(t *testing.T) {
	bots := []*Bot{
		{Name: "googlebot", UA: "Googlebot"},
		{Name: "bingbot", UA: "Bingbot"},
		{Name: "emptyua", UA: ""},
	}

	index := buildUAIndex(bots)

	// Should have entries for 'G' and 'B'
	if len(index) != 2 {
		t.Errorf("Expected index with 2 entries, got %d", len(index))
	}

	// Check G entry
	gBots, ok := index['G']
	if !ok {
		t.Error("Expected index['G'] to exist")
	}
	if len(gBots) != 1 || gBots[0].Name != "googlebot" {
		t.Errorf("Expected googlebot in index['G'], got %v", gBots)
	}

	// Check B entry
	bBots, ok := index['B']
	if !ok {
		t.Error("Expected index['B'] to exist")
	}
	if len(bBots) != 1 || bBots[0].Name != "bingbot" {
		t.Errorf("Expected bingbot in index['B'], got %v", bBots)
	}

	// Empty UA should not be in index
	if _, ok := index[0]; ok {
		t.Error("Empty UA should not be in index")
	}
}

func TestLoadBotInvalidFile(t *testing.T) {
	// Test loading a non-existent file
	_, err := loadBot("/nonexistent/path/bot.yaml")
	if err == nil {
		t.Error("Expected error loading non-existent file")
	}
}

func TestLoadBotInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(path, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, err := loadBot(path)
	if err == nil {
		t.Error("Expected error parsing invalid YAML")
	}
}

func TestLoadBotMissingName(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "missing-name.yaml")
	// Config without 'name' field
	config := `ua: "TestBot"
custom:
  - "192.168.1.0/24"
`
	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	bot, err := loadBot(path)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if bot != nil {
		t.Error("Expected nil bot for missing name")
	}
}

func TestLoadBotInvalidCIDR(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "testbot.yaml")
	config := `name: testbot
ua: "TestBot"
custom:
  - "invalid-cidr"
  - "192.168.1.0/24"
`
	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	bot, err := loadBot(path)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have parsed the valid CIDR
	prefixes := bot.custom.Load()
	if len(*prefixes) != 1 {
		t.Errorf("Expected 1 valid CIDR, got %d", len(*prefixes))
	}
}

func TestParseCIDRs(t *testing.T) {
	tests := []struct {
		input  []string
		expect int
	}{
		{[]string{"192.168.1.0/24"}, 1},
		{[]string{"10.0.0.0/8"}, 1},
		{[]string{"invalid"}, 0},
		{[]string{""}, 0},
		{[]string{"192.168.1.0/24", "invalid", "10.0.0.0/8"}, 2},
	}

	for _, tt := range tests {
		result := parseCIDRs(tt.input)
		if len(result) != tt.expect {
			t.Errorf("parseCIDRs(%v) = %d prefixes, want %d", tt.input, len(result), tt.expect)
		}
	}
}

func TestConfigOptions(t *testing.T) {
	// Test WithRoot
	cfg := &Config{}
	WithRoot("/custom/bots")(cfg)
	if cfg.Root != "/custom/bots" {
		t.Errorf("WithRoot failed: got %q, want %q", cfg.Root, "/custom/bots")
	}

	// Test WithFailLimit
	WithFailLimit(500)(cfg)
	if cfg.FailLimit != 500 {
		t.Errorf("WithFailLimit failed: got %d, want 500", cfg.FailLimit)
	}

	// Test WithClassifyUA
	WithClassifyUA()(cfg)
	if !cfg.ClassifyUA {
		t.Error("WithClassifyUA failed: ClassifyUA should be true")
	}
}
