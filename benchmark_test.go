package knownbots

import (
	"os"
	"path/filepath"
	"testing"
)

func init() {
	EnableLog = false
}

func setupBenchValidator(b *testing.B, botCount int) *Validator {
	tmpDir := b.TempDir()
	confDir := filepath.Join(tmpDir, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		b.Fatal(err)
	}

	botNames := []string{
		"TestBot1", "TestBot2", "TestBot3", "TestBot4", "TestBot5",
		"TestBot6", "TestBot7", "TestBot8", "TestBot9", "TestBot10",
		"TestBot11", "TestBot12", "TestBot13", "TestBot14", "TestBot15",
		"TestBot16", "TestBot17", "TestBot18", "TestBot19", "TestBot20",
		"TestBot21", "TestBot22", "TestBot23", "TestBot24", "TestBot25",
		"TestBot26", "TestBot27", "TestBot28", "TestBot29", "TestBot30",
		"TestBot31", "TestBot32", "TestBot33", "TestBot34", "TestBot35",
		"TestBot36", "TestBot37", "TestBot38", "TestBot39", "TestBot40",
	}

	for i := 0; i < botCount && i < len(botNames); i++ {
		botName := botNames[i]
		content := "name: " + botName + "\nua: \"" + botName + "\"\ncustom:\n  - \"192.168.1.0/24\"\ndomains:\n  - \"example.com\"\n"
		path := filepath.Join(confDir, botName+".yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	v, err := New(WithRoot(tmpDir))
	if err != nil {
		b.Fatal(err)
	}
	return v
}

func BenchmarkFindBotByUA_Hit_First(b *testing.B) {
	v := setupBenchValidator(b, 40)
	defer v.Close()
	ua := "Mozilla/5.0 (compatible; TestBot1/2.1; +http://example.com/bot.html)"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if v.findBotByUA(ua) == nil {
				b.Fatal("expected to find Googlebot")
			}
		}
	})
}

func BenchmarkFindBotByUA_Hit_Middle(b *testing.B) {
	v := setupBenchValidator(b, 40)
	defer v.Close()
	ua := "Mozilla/5.0 (compatible; TestBot2/1.0)"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if v.findBotByUA(ua) == nil {
				b.Fatal("expected to find DuckDuckBot")
			}
		}
	})
}

func BenchmarkFindBotByUA_Hit_Last(b *testing.B) {
	v := setupBenchValidator(b, 40)
	defer v.Close()
	ua := "Mozilla/5.0 (compatible; TestBot40/1.0)"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if v.findBotByUA(ua) == nil {
				b.Fatal("expected to find SiteAuditBot")
			}
		}
	})
}

func BenchmarkFindBotByUA_Miss(b *testing.B) {
	v := setupBenchValidator(b, 40)
	defer v.Close()
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if v.findBotByUA(ua) != nil {
				b.Fatal("expected not to find any bot")
			}
		}
	})
}

func BenchmarkFindBotByUA_CaseSensitive(b *testing.B) {
	v := setupBenchValidator(b, 40)
	defer v.Close()
	ua := "Mozilla/5.0 (compatible; TestBot1/2.1)"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if v.findBotByUA(ua) == nil {
				b.Fatal("expected to find Googlebot")
			}
		}
	})
}

func BenchmarkValidate_KnownBot_IPHit(b *testing.B) {
	v := setupBenchValidator(b, 40)
	defer v.Close()
	ua := "Mozilla/5.0 (compatible; TestBot1/2.1)"
	ip := "192.168.1.100"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := v.Validate(ua, ip)
			if result.Status != StatusVerified {
				b.Fatalf("expected verified, got %d", result.Status)
			}
		}
	})
}

func BenchmarkValidate_Browser(b *testing.B) {
	v := setupBenchValidator(b, 40)
	defer v.Close()
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	ip := "192.168.1.1"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := v.Validate(ua, ip)
			if result.IsBot {
				b.Fatal("expected IsBot=false for browser")
			}
		}
	})
}

func BenchmarkContainsWord(b *testing.B) {
	text := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	word := "Googlebot"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !containsWord(text, word) {
			b.Fatal("expected match")
		}
	}
}
