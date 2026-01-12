package knownbots

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRDNS_GetSet(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.txt")

	// Create cache
	rdnscache, err := NewRDNS(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	// Test Get on empty cache
	_, ok := rdnscache.Get("192.168.1.1")
	if ok {
		t.Error("expected Get to return false for empty cache")
	}

	// Test Set
	rdnscache.Set("192.168.1.1", "example.com")
	rdnscache.Set("192.168.1.2", "test.com")

	// Test Get after Set
	val, ok := rdnscache.Get("192.168.1.1")
	if !ok {
		t.Error("expected Get to return true after Set")
	}
	if val != "example.com" {
		t.Errorf("expected 'example.com', got '%s'", val)
	}

	// Test Size
	if rdnscache.Size() != 2 {
		t.Errorf("expected size 2, got %d", rdnscache.Size())
	}

	// Test Set duplicate key (should not increase size)
	rdnscache.Set("192.168.1.1", "newdomain.com")
	if rdnscache.Size() != 2 {
		t.Errorf("expected size 2 after duplicate Set, got %d", rdnscache.Size())
	}
}

func TestRDNS_Persist(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.txt")

	// Create cache and add entries
	rdnscache, err := NewRDNS(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	rdnscache.Set("192.168.1.1", "example.com")
	rdnscache.Set("192.168.1.2", "test.com")

	// Persist to file
	if err := rdnscache.Persist(); err != nil {
		t.Fatal(err)
	}

	// Create new cache from same file
	rdnscache2, err := NewRDNS(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify entries were persisted
	val, ok := rdnscache2.Get("192.168.1.1")
	if !ok {
		t.Error("expected to find persisted entry")
	}
	if val != "example.com" {
		t.Errorf("expected 'example.com', got '%s'", val)
	}
}

func TestRDNS_Prune(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.txt")

	rdnscache, err := NewRDNS(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	// Add entries with different domains
	rdnscache.Set("192.168.1.1", "example.com")     // matches example.com
	rdnscache.Set("192.168.1.2", "test.com")        // doesn't match
	rdnscache.Set("192.168.1.3", "sub.example.com") // matches example.com

	// Prune keeping only example.com domains
	rdnscache.Prune([]string{"example.com"})

	// Check sizes
	if rdnscache.Size() != 2 {
		t.Errorf("expected size 2 after prune, got %d", rdnscache.Size())
	}

	// Check that matching entry is preserved
	_, ok := rdnscache.Get("192.168.1.1")
	if !ok {
		t.Error("expected example.com entry to be preserved")
	}

	// Check that non-matching entry was removed
	_, ok = rdnscache.Get("192.168.1.2")
	if ok {
		t.Error("expected test.com entry to be pruned")
	}
}

func TestRDNS_Close(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.txt")

	rdnscache, err := NewRDNS(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	// Close should be a no-op and return nil
	if err := rdnscache.Close(); err != nil {
		t.Errorf("expected Close to return nil, got %v", err)
	}
}

func TestLRU_AddContains(t *testing.T) {
	lru := NewLRU(3)

	// Test Contains on empty LRU
	if lru.Contains("key1") {
		t.Error("expected Contains to return false for empty LRU")
	}

	// Add entries
	lru.Add("key1")
	lru.Add("key2")
	lru.Add("key3")

	// Test Contains after Add
	if !lru.Contains("key1") {
		t.Error("expected Contains to return true after Add")
	}
	if !lru.Contains("key2") {
		t.Error("expected Contains to return true after Add")
	}

	// Test LRU eviction - add 4th item, key1 should be evicted (oldest)
	lru.Add("key4")

	// key1 should be evicted (least recently used)
	if lru.Contains("key1") {
		t.Error("expected key1 to be evicted after adding key4")
	}

	// key2, key3, key4 should still be present
	if !lru.Contains("key2") {
		t.Error("expected key2 to be present")
	}
	if !lru.Contains("key3") {
		t.Error("expected key3 to be present")
	}
	if !lru.Contains("key4") {
		t.Error("expected key4 to be present")
	}
}

func TestLRU_Eviction(t *testing.T) {
	limit := 3
	lru := NewLRU(limit)

	// Fill up to limit
	lru.Add("key1")
	lru.Add("key2")
	lru.Add("key3")

	// Add one more to trigger eviction
	lru.Add("key4")

	// key1 should be evicted (least recently used)
	if lru.Contains("key1") {
		t.Error("expected key1 to be evicted")
	}

	// key2, key3, key4 should still be present
	if !lru.Contains("key2") {
		t.Error("expected key2 to be present")
	}
	if !lru.Contains("key3") {
		t.Error("expected key3 to be present")
	}
	if !lru.Contains("key4") {
		t.Error("expected key4 to be present")
	}
}

func TestWriteIPs(t *testing.T) {
	tmpDir := t.TempDir()
	ipPath := filepath.Join(tmpDir, "ips.txt")

	// Test writing empty slice
	writeIPs(ipPath, nil)

	// File should be created (possibly empty)
	if _, err := os.Stat(ipPath); err != nil {
		t.Error("expected file to be created")
	}

	// Test writing prefixes
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("192.168.1.0/24"),
		netip.MustParsePrefix("10.0.0.0/8"),
	}
	writeIPs(ipPath, prefixes)

	// Read and verify content
	data, err := os.ReadFile(ipPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "192.168.1.0/24") {
		t.Error("expected 192.168.1.0/24 in file")
	}
	if !strings.Contains(content, "10.0.0.0/8") {
		t.Error("expected 10.0.0.0/8 in file")
	}
}

func TestWriteIPs_Error(t *testing.T) {
	// Test writeIPs with invalid directory (should not panic)
	tmpDir := t.TempDir()
	ipPath := filepath.Join(tmpDir, "nonexistent", "path", "ips.txt")

	prefixes := []netip.Prefix{
		netip.MustParsePrefix("192.168.1.0/24"),
	}
	// Should not panic, just return silently
	writeIPs(ipPath, prefixes)
}

func TestBotContainsIP(t *testing.T) {
	tree := NewIPTree()
	tree.Add(netip.MustParsePrefix("192.168.1.0/24"))
	tree.Add(netip.MustParsePrefix("10.0.0.0/8"))

	bot := &Bot{
		Name: "testbot",
		UA:   "TestBot",
		ips:  &atomic.Pointer[IPTree]{},
	}
	bot.ips.Store(tree)

	// Test contained IP
	if !bot.ContainsIP("192.168.1.100") {
		t.Error("expected 192.168.1.100 to be contained in 192.168.1.0/24")
	}
	if !bot.ContainsIP("10.0.0.1") {
		t.Error("expected 10.0.0.1 to be contained in 10.0.0.0/8")
	}

	// Test non-contained IP
	if bot.ContainsIP("172.16.0.1") {
		t.Error("expected 172.16.0.1 to NOT be contained")
	}

	// Test invalid IP
	if bot.ContainsIP("invalid-ip") {
		t.Error("expected invalid IP to return false")
	}
}

func TestValidator_Reload(t *testing.T) {
	// Create a temporary directory with bot configs
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a bot config
	configContent := `name: reloadbot
ua: "ReloadBot"
custom:
  - "192.168.1.0/24"
`
	if err := os.WriteFile(filepath.Join(confDir, "reloadbot.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create validator with the temp directory
	EnableLog = false
	defer func() { EnableLog = true }()

	v, err := New(WithRoot(tmpDir))
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	// Verify bot is loaded
	bot := v.findBotByUA("Mozilla/5.0 (compatible; ReloadBot/1.0)")
	if bot == nil {
		t.Fatal("expected to find reloadbot after initial load")
	}

	// Add another bot config
	configContent2 := `name: newbot
ua: "NewBot"
custom:
  - "10.0.0.0/8"
`
	if err := os.WriteFile(filepath.Join(confDir, "newbot.yaml"), []byte(configContent2), 0644); err != nil {
		t.Fatal(err)
	}

	// Reload
	if err := v.Reload(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Verify both bots are loaded
	bot1 := v.findBotByUA("Mozilla/5.0 (compatible; ReloadBot/1.0)")
	if bot1 == nil {
		t.Error("expected to find reloadbot after reload")
	}

	bot2 := v.findBotByUA("Mozilla/5.0 (compatible; NewBot/1.0)")
	if bot2 == nil {
		t.Error("expected to find newbot after reload")
	}
}
