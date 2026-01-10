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
		"Googlebot", "Bingbot", "Slurp", "DuckDuckBot", "Baiduspider",
		"YandexBot", "Sogou", "facebookexternalhit", "Twitterbot", "LinkedInBot",
		"TelegramBot", "Discordbot", "Slackbot", "WhatsApp", "Applebot",
		"AhrefsBot", "SemrushBot", "MJ12bot", "ScreamingFrog", "PetalBot",
		"Bytespider", "SeznamBot", "Exabot", "DotBot", "rogerbot",
		"archive_bot", "ia_archiver", "BLEXBot", "GPTBot", "Claude-Web",
		"anthropic-ai", "Pinterestbot", "facebookcatalog", "Tumblr", "Mastodon",
		"PiplBot", "SentiBot", "DataForSeoBot", "MegaIndex", "SiteAuditBot",
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
	ua := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

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
	ua := "Mozilla/5.0 (compatible; DuckDuckBot/1.0)"

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
	ua := "Mozilla/5.0 (compatible; SiteAuditBot/1.0)"

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

func BenchmarkFindBotByUA_CaseInsensitive(b *testing.B) {
	v := setupBenchValidator(b, 40)
	ua := "Mozilla/5.0 (compatible; googlebot/2.1)"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if v.findBotByUA(ua) == nil {
				b.Fatal("expected to find googlebot (lowercase)")
			}
		}
	})
}

func BenchmarkValidate_KnownBot_IPHit(b *testing.B) {
	v := setupBenchValidator(b, 40)
	ua := "Mozilla/5.0 (compatible; Googlebot/2.1)"
	ip := "192.168.1.100"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := v.Validate(ua, ip)
			if result.Status != StatusVerified {
				b.Fatalf("expected verified, got %s", result.Status)
			}
		}
	})
}

func BenchmarkValidate_Browser(b *testing.B) {
	v := setupBenchValidator(b, 40)
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
