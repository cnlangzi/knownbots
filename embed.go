package knownbots

import (
	"embed"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

//go:embed bots/conf.d/*.yaml
var embeddedBots embed.FS

// loadEmbedded loads all built-in bot configurations from the embedded filesystem.
func loadEmbedded() (map[string]*Bot, error) {
	bots := make(map[string]*Bot)

	err := fs.WalkDir(embeddedBots, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" {
			return nil
		}

		data, err := embeddedBots.ReadFile(path)
		if err != nil {
			return err
		}

		bot, err := parseBotConfig(data, path)
		if err != nil {
			return err
		}
		if bot == nil {
			return nil
		}

		bots[bot.Name] = bot
		return nil
	})

	if err != nil {
		return nil, err
	}

	return bots, nil
}

// parseBotConfig parses bot configuration from YAML data.
func parseBotConfig(data []byte, filename string) (*Bot, error) {
	var cfg botConfig

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Validate required Name field
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
	customValue := &atomic.Pointer[[]IPPrefix]{}
	customValue.Store(&customNets)

	return &Bot{
		Name:    cfg.Name,
		Kind:    cfg.Kind,
		Parser:  parser,
		UA:      cfg.UA,
		URLs:    cfg.URLs,
		custom:  customValue,
		Domains: cfg.Domains,
		RDNS:    cfg.RDNS,
	}, nil
}
