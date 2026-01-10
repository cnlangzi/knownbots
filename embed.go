package knownbots

import (
	"embed"
	"io/fs"
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

		bot, err := parseBotConfig(data)
		if err != nil {
			return err
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
func parseBotConfig(data []byte) (*Bot, error) {
	var tmp struct {
		Name    string   `yaml:"name"`
		Kind    BotKind  `yaml:"kind"`
		Parser  string   `yaml:"parser"`
		UA      string   `yaml:"ua"`
		URLs    []string `yaml:"urls"`
		Custom  []string `yaml:"custom"`
		Domains []string `yaml:"domains"`
		RDNS    bool     `yaml:"rdns"`
	}

	if err := yaml.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}

	parser := tmp.Parser
	if parser == "" {
		parser = tmp.Name
	}

	customNets := parseCIDRs(tmp.Custom)
	customValue := &atomic.Value{}
	customValue.Store(customNets)

	return &Bot{
		Name:    tmp.Name,
		Kind:    tmp.Kind,
		Parser:  parser,
		UA:      tmp.UA,
		URLs:    tmp.URLs,
		custom:  customValue,
		Domains: tmp.Domains,
		RDNS:    tmp.RDNS,
	}, nil
}
