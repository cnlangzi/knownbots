package knownbots

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
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

		bot, err := buildBot(data, path)
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
