package knownbots

import (
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/cnlangzi/knownbots/asn"
	"github.com/kentik/patricia"
	"github.com/kentik/patricia/uint_tree"
	"gopkg.in/yaml.v3"
)

type IPPrefix = netip.Prefix

type IPTree struct {
	v4 *uint_tree.TreeV4
	v6 *uint_tree.TreeV6
}

func NewIPTree() *IPTree {
	return &IPTree{
		v4: uint_tree.NewTreeV4(),
		v6: uint_tree.NewTreeV6(),
	}
}

func (t *IPTree) Add(prefix netip.Prefix) {
	addr := prefix.Addr()
	if addr.Is4() {
		patriciaAddr, _, _ := patricia.ParseFromNetIPPrefix(prefix)
		if patriciaAddr != nil {
			t.v4.Add(*patriciaAddr, 1, nil)
		}
	} else {
		_, patriciaAddr, _ := patricia.ParseFromNetIPPrefix(prefix)
		if patriciaAddr != nil {
			t.v6.Add(*patriciaAddr, 1, nil)
		}
	}
}

func (t *IPTree) Contains(ip netip.Addr) bool {
	patriciaAddrV4, patriciaAddrV6, _ := patricia.ParseFromNetIPAddr(ip)
	if ip.Is4() && patriciaAddrV4 != nil {
		found, _ := t.v4.FindDeepestTag(*patriciaAddrV4)
		return found
	}
	if patriciaAddrV6 != nil {
		found, _ := t.v6.FindDeepestTag(*patriciaAddrV6)
		return found
	}
	return false
}

func (t *IPTree) Count() int {
	return t.v4.CountTags() + t.v6.CountTags()
}

type BotKind string

const (
	KindSearchEngine BotKind = "SearchEngine"
	KindSocialMedia  BotKind = "SocialMedia"
	KindAITraining   BotKind = "AITraining"
	KindAIAssist     BotKind = "AIAssist"
	KindAIMixed      BotKind = "AIMixed"
	KindSEO          BotKind = "SEO"
	KindMonitor      BotKind = "Monitor"
	KindSecurity     BotKind = "Security"
	KindScraper      BotKind = "Scraper"
	KindUnknown      BotKind = "Unknown"
)

type botConfig struct {
	Name    string   `yaml:"name"`
	Kind    BotKind  `yaml:"kind"`
	Parser  string   `yaml:"parser"`
	UA      string   `yaml:"ua"`
	URLs    []string `yaml:"urls"`
	Custom  []string `yaml:"custom"`
	ASN     []int    `yaml:"asn"`
	Domains []string `yaml:"domains"`
	RDNS    bool     `yaml:"rdns"`
}

type Bot struct {
	Name    string   `yaml:"name"`
	Kind    BotKind  `yaml:"kind"`
	Parser  string   `yaml:"parser"`
	UA      string   `yaml:"ua"`
	URLs    []string `yaml:"urls"`
	ips     *atomic.Pointer[IPTree]
	asns    *atomic.Pointer[ASN]
	ASN     []int    `yaml:"asn"`
	Domains []string `yaml:"domains"`
	RDNS    bool     `yaml:"rdns"`
	rdns    *RDNS
	fail    *LRU
}

func (b *Bot) storePrefixes(prefixes []netip.Prefix) {
	if len(prefixes) == 0 {
		return
	}
	if b.ips == nil {
		b.ips = &atomic.Pointer[IPTree]{}
	}

	tree := NewIPTree()
	for _, prefix := range prefixes {
		tree.Add(prefix)
	}
	b.ips.Store(tree)
}

func (b *Bot) initIPs(path string) {
	if len(b.URLs) > 0 {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if EnableLog {
			log.Printf("initIPs: failed to read IP prefixes from %q: %v", path, err)
		}
		return
	}

	var prefixes []netip.Prefix
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(line)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	b.storePrefixes(prefixes)
}

func (b *Bot) refreshIPs(httpClient *http.Client, root string) {
	prefixes := downloadIPs(httpClient, b)
	if len(prefixes) == 0 {
		return
	}

	b.storePrefixes(prefixes)
	writeIPs(filepath.Join(root, b.Name, "ips.txt"), prefixes)
}

func (b *Bot) initRDNS(path string) {
	if b.RDNS {
		rdnscache, err := NewRDNS(path)
		if err != nil {
			if EnableLog {
				log.Printf("[knownbots] failed to create rdns cache for %s: %v", path, err)
			}
			return
		}
		b.rdns = rdnscache
	}

}

func (b *Bot) refreshRDNS() {
	if b.RDNS && b.rdns != nil {
		rdnscache := b.rdns
		rdnscache.Prune(b.Domains)

		if err := rdnscache.Persist(); err != nil {
			if EnableLog {
				log.Printf("[knownbots] failed to persist cache for %s: %v", b.Name, err)
			}
		}
	}
}

func (b *Bot) initASN(store *asn.Store) {
	if len(b.ASN) == 0 {
		return
	}

	if b.asns == nil {
		b.asns = &atomic.Pointer[ASN]{}
	}

	cache := NewASN()
	for _, asnNum := range b.ASN {
		prefixes := store.Load(b.Name, asnNum)
		if len(prefixes) == 0 {
			continue
		}
		cache.Add(asnNum, prefixes)
	}
	b.asns.Store(cache)
}

func (b *Bot) refreshASN(store *asn.Store) {
	if len(b.ASN) == 0 {
		return
	}

	newASN := NewASN()
	for _, asnNum := range b.ASN {
		prefixes := store.Refresh(b.Name, asnNum)
		if len(prefixes) == 0 {
			continue
		}
		newASN.Add(asnNum, prefixes)
	}

	if b.asns == nil {
		b.asns = &atomic.Pointer[ASN]{}
	}
	b.asns.Store(newASN)
}

func Load(dir string) ([]*Bot, error) {
	embedded, err := loadEmbedded()
	if err != nil {
		return nil, err
	}

	customDir := filepath.Join(dir, "conf.d")
	entries, err := os.ReadDir(customDir)
	if err != nil {
		if os.IsNotExist(err) {
			bots := make([]*Bot, 0, len(embedded))
			for _, bot := range embedded {
				bots = append(bots, bot)
			}
			return bots, nil
		}
		return nil, err
	}

	bots := embedded

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(customDir, entry.Name())
		bot, err := loadBot(path)
		if err != nil {
			return nil, err
		}
		if bot == nil {
			continue
		}

		if _, exists := embedded[bot.Name]; exists {
			if EnableLog {
				log.Printf("[knownbots] custom config %q overrides built-in bot", bot.Name)
			}
		}
		bots[bot.Name] = bot
	}

	result := make([]*Bot, 0, len(bots))
	for _, bot := range bots {
		result = append(result, bot)
	}

	return result, nil
}

func loadBot(path string) (*Bot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return buildBot(data, path)
}

func buildBot(data []byte, filename string) (*Bot, error) {
	var cfg botConfig

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Name == "" {
		if EnableLog {
			log.Printf("[knownbots] skip %q: missing required 'name' field", filename)
		}
		return nil, nil
	}

	parser := cfg.Parser
	if parser == "" {
		parser = cfg.Name
	}

	customNets := parseCIDRs(cfg.Custom)
	ips := &atomic.Pointer[IPTree]{}
	if len(customNets) > 0 {
		tree := NewIPTree()
		for _, prefix := range customNets {
			tree.Add(prefix)
		}
		if tree.Count() > 0 {
			ips.Store(tree)
		}
	}

	return &Bot{
		Name:    cfg.Name,
		Kind:    cfg.Kind,
		Parser:  parser,
		UA:      cfg.UA,
		URLs:    cfg.URLs,
		ips:     ips,
		asns:    nil,
		ASN:     cfg.ASN,
		Domains: cfg.Domains,
		RDNS:    cfg.RDNS,
		rdns:    nil,
		fail:    nil,
	}, nil
}

func parseCIDRs(cidrs []string) []IPPrefix {
	var nets []IPPrefix
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil {
			nets = append(nets, prefix)
		}
	}
	return nets
}

func (b *Bot) ContainsIP(ipStr string) bool {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}

	if b.ips != nil {
		tree := b.ips.Load()
		if tree != nil && tree.Contains(ip) {
			return true
		}
	}
	return false
}

func (b *Bot) VerifyRDNS(ipStr string) ResultStatus {
	if b.fail.Contains(ipStr) {
		return StatusFailed
	}

	if hostname, ok := b.rdns.Get(ipStr); ok {
		if matchDomain(hostname, b.Domains) {
			return StatusVerified
		}
		return StatusFailed
	}

	names, err := net.LookupAddr(ipStr)
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			b.fail.Add(ipStr)
			return StatusFailed
		}
		return StatusPending
	}
	if len(names) == 0 {
		b.fail.Add(ipStr)
		return StatusFailed
	}

	hostname := strings.TrimSuffix(names[0], ".")
	if matchDomain(hostname, b.Domains) {
		b.rdns.Set(ipStr, hostname)
		return StatusVerified
	}

	b.fail.Add(ipStr)
	return StatusFailed
}
