package parser

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func fetchFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func TestGoogleParserWithRealFormat(t *testing.T) {
	p := &GoogleParser{}
	testInput := `{"prefixes": [{"ipv4Prefix": "66.249.64.0/19", "ipv6Prefix": "2001:db8::/32"}, {"ipv4Prefix": "142.250.0.0/15"}]}`
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 prefixes, got %d", len(result))
	}
}

func TestOpenAIParserWithRealFormat(t *testing.T) {
	p := &OpenAIStyleParser{}
	testInput := `{"prefixes": [{"prefix": "132.196.86.0/24"}, {"prefix": "172.182.202.0/25"}]}`
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestGitHubParserWithRealFormat(t *testing.T) {
	p := &GitHubParser{}
	testInput := `{"hooks": ["192.30.252.0/22"], "web": ["192.30.252.0/22"], "api": ["192.30.252.0/22"]}`
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestStripeParserWithRealFormat(t *testing.T) {
	p := &StripeParser{}
	testInput := `{"WEBHOOKS": ["3.18.12.63", "3.130.192.231", "13.235.14.237"]}`
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 prefixes, got %d", len(result))
	}
}

func TestTxtParserWithRealFormat(t *testing.T) {
	p := &TxtParser{}
	testInput := "172.217.0.0/16\n142.250.0.0/15\n192.178.0.0/15"
	result, err := p.Parse(strings.NewReader(testInput))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 prefixes, got %d", len(result))
	}
}

func TestIntegration_GoogleBot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://developers.google.com/static/search/apis/ipranges/googlebot.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("GoogleBot IP list should not be empty")
	}
	t.Logf("GoogleBot: parsed %d prefixes", len(result))
}

func TestIntegration_Bingbot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://www.bing.com/toolbox/bingbot.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("Bingbot IP list should not be empty")
	}
	t.Logf("Bingbot: parsed %d prefixes", len(result))
}

func TestIntegration_GPTBot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://openai.com/gptbot.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("GPTBot IP list should not be empty")
	}
	t.Logf("GPTBot: parsed %d prefixes", len(result))
}

func TestIntegration_GitHub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GitHubParser{}
	data, err := fetchFromURL("https://api.github.com/meta")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("GitHub IP list should not be empty")
	}
	t.Logf("GitHub: parsed %d prefixes", len(result))
}

func TestIntegration_Stripe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &StripeParser{}
	data, err := fetchFromURL("https://stripe.com/files/ips/ips_webhooks.json")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("Stripe webhook IP list should not be empty")
	}
	t.Logf("Stripe: parsed %d IPs", len(result))
}

func TestIntegration_UptimeRobot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &TxtParser{}
	data, err := fetchFromURL("https://uptimerobot.com/inc/files/ips/IPv4.txt")
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("UptimeRobot IP list should not be empty")
	}
	t.Logf("UptimeRobot: parsed %d prefixes", len(result))
}

func TestIntegration_Applebot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := &GoogleParser{}
	data, err := fetchFromURL("https://search.developer.apple.com/applebot.json")
	if err != nil {
		t.Skipf("network unavailable, skipping: %v", err)
	}
	result, err := p.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(result) == 0 {
		t.Error("Applebot IP list should not be empty")
	}
	t.Logf("Applebot: parsed %d prefixes", len(result))
}
