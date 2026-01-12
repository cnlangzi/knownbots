package asn

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
)

// Store handles ASN data fetching and caching.
type Store struct {
	cacheDir string
	fetchers []Fetcher
}

// NewStore creates a new ASN store.
func NewStore(cacheDir string) *Store {
	return &Store{
		cacheDir: cacheDir,
		fetchers: []Fetcher{
			NewRIPE(),
			NewRouteViews(),
			NewBGPHE(),
		},
	}
}

// Get returns prefixes for a given ASN from cache or fetches them.
func (s *Store) Refresh(botName string, asn int) []netip.Prefix {
	prefixes, err := s.fetch(asn)
	if err != nil {
		return nil
	}

	_ = s.save(botName, asn, prefixes)

	return prefixes
}

func (s *Store) Load(botName string, asn int) []netip.Prefix {
	path := filepath.Join(s.cacheDir, botName, fmt.Sprintf("AS%d.json", asn))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var prefixStrings []string
	if err := json.Unmarshal(data, &prefixStrings); err != nil {
		return nil
	}

	prefixes := make([]netip.Prefix, 0, len(prefixStrings))
	for _, p := range prefixStrings {
		prefix, err := netip.ParsePrefix(p)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

func (s *Store) save(botName string, asn int, prefixes []netip.Prefix) error {
	path := filepath.Join(s.cacheDir, botName, fmt.Sprintf("AS%d.json", asn))

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	prefixStrings := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		prefixStrings = append(prefixStrings, p.String())
	}

	data, err := json.MarshalIndent(prefixStrings, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *Store) fetch(asn int) ([]netip.Prefix, error) {
	var lastErr error
	for _, fetcher := range s.fetchers {
		prefixes, err := fetcher.Fetch(asn)
		if err != nil {
			lastErr = err
			continue
		}
		if len(prefixes) > 0 {
			return prefixes, nil
		}
	}

	if lastErr == nil {
		return nil, fmt.Errorf("no prefixes found for ASN %d from any fetcher", asn)
	}

	return nil, fmt.Errorf("all fetchers failed for ASN %d: %w", asn, lastErr)
}
