package knownbots

import (
	"bytes"
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
// It returns a map of bot name to Bot for quick lookup.
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
		if ext != ".yaml" && ext != ".yml" {
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

	// Use bot name as default parser if not specified
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

// embeddedBotsFS returns the embedded bots filesystem for external use.
func embeddedBotsFS() embed.FS {
	return embeddedBots
}

// EmbeddedBotNames returns the list of all built-in bot names.
// Useful for debugging or documentation.
func EmbeddedBotNames() []string {
	bots, _ := loadEmbedded()
	names := make([]string, 0, len(bots))
	for name := range bots {
		names = append(names, name)
	}
	return names
}

// LoadEmbeddedBot loads a single built-in bot configuration by name.
// Returns nil if the bot is not found.
func LoadEmbeddedBot(name string) *Bot {
	bots, _ := loadEmbedded()
	return bots[name]
}

// IsEmbeddedBot returns true if the given bot name is a built-in bot.
func IsEmbeddedBot(name string) bool {
	bots, _ := loadEmbedded()
	_, ok := bots[name]
	return ok
}

// EmbeddedBotCount returns the number of built-in bots.
func EmbeddedBotCount() int {
	bots, _ := loadEmbedded()
	return len(bots)
}

// ReadEmbeddedBotConfig reads the raw YAML configuration for a built-in bot.
// Returns the raw bytes of the config file.
func ReadEmbeddedBotConfig(name string) ([]byte, error) {
	bots, err := loadEmbedded()
	if err != nil {
		return nil, err
	}

	bot, ok := bots[name]
	if !ok {
		return nil, nil
	}

	// Reconstruct YAML from the bot struct
	customSlice := parseCustomToString(bot.custom.Load().([]IPPrefix))

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	defer encoder.Close()

	err = encoder.Encode(map[string]interface{}{
		"kind":    bot.Kind,
		"name":    bot.Name,
		"parser":  bot.Parser,
		"ua":      bot.UA,
		"urls":    bot.URLs,
		"custom":  customSlice,
		"domains": bot.Domains,
		"rdns":    bot.RDNS,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// parseCustomToString converts custom networks back to string format.
func parseCustomToString(nets []IPPrefix) []string {
	var result []string
	for _, net := range nets {
		result = append(result, net.String())
	}
	return result
}
