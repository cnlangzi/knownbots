package knownbots

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestValidateInvalidIP(t *testing.T) {
	tmpDir := t.TempDir()

	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("Failed to create conf.d: %v", err)
	}

	configContent := `kind: SearchEngine
name: testbot
ua: "TestBot"
asn:
  - 15169
custom:
  - "192.168.1.0/24"
domains:
  - "testbot.example.com"
rdns: true
`
	configPath := filepath.Join(confDir, "testbot.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	v, err := New(WithRoot(tmpDir))
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	defer v.Close()

	invalidIPs := []string{
		"invalid-ip",
		"999.999.999.999",
		"not.an.ip.address",
		"",
		":::",
		"256.256.256.256",
		"2001:db8::gggg",
		"192.168.1",
		"192.168.1.1.1",
	}

	for _, ip := range invalidIPs {
		t.Run(ip, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Validate panicked with invalid IP %q: %v", ip, r)
				}
			}()

			result := v.Validate("TestBot/1.0", ip)

			if result.Status == StatusVerified {
				t.Errorf("invalid IP %q should not be verified", ip)
			}
		})
	}
}

func TestVerifyIPWithASN(t *testing.T) {
	tmpDir := t.TempDir()

	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("Failed to create conf.d: %v", err)
	}

	configContent := `kind: SearchEngine
name: testbot
ua: "TestBot"
asn:
  - 15169
`
	configPath := filepath.Join(confDir, "testbot.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	v, err := New(WithRoot(tmpDir))
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	defer v.Close()

	bots := v.bots.Load()
	if len(*bots) == 0 {
		t.Fatal("No bots loaded")
	}

	var testBot *Bot
	for _, bot := range *bots {
		if bot.Name == "testbot" {
			testBot = bot
			break
		}
	}

	if testBot == nil {
		t.Fatal("testbot not found")
	}

	asnCache := NewASN()
	testBot.asns = &atomic.Pointer[ASN]{}
	testBot.asns.Store(asnCache)

	testCases := []struct {
		name       string
		ip         string
		shouldFail bool
	}{
		{"valid IPv4", "8.8.8.8", false},
		{"valid IPv6", "2001:db8::1", false},
		{"invalid IP", "invalid-ip", true},
		{"malformed IPv4", "999.999.999.999", true},
		{"malformed IPv6", "2001:db8::gggg", true},
		{"empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("verifyIP panicked with %q: %v", tc.ip, r)
				}
			}()

			result := v.verifyIP(testBot, tc.ip)

			if tc.shouldFail && result.Status == StatusVerified {
				t.Errorf("expected %q to fail verification, got %v", tc.ip, result.Status)
			}
		})
	}
}
