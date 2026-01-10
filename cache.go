package knownbots

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// Cache provides thread-safe reverse DNS lookup caching.
// It stores successfully verified IP→hostname mappings (persistent).
type Cache struct {
	valid atomic.Pointer[map[string]string] // *map[string]string (read-heavy, lock-free reads)
	file  string
}

// NewCache creates a new Cache instance.
func NewCache(filePath string) (*Cache, error) {
	c := &Cache{
		file: filePath,
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	// Load existing cache from file
	if err := c.loadFromFile(); err != nil {
		return nil, err
	}

	return c, nil
}

// Get retrieves a value from the cache.
func (c *Cache) Get(key string) (string, bool) {
	m := c.valid.Load()
	val, ok := (*m)[key]
	return val, ok
}

// Set stores a successful lookup result in the cache.
func (c *Cache) Set(key, value string) {
	old := c.valid.Load()
	// Check if already exists (fast path)
	if _, ok := (*old)[key]; ok {
		return
	}
	// Create new map with the entry
	newMap := make(map[string]string, len(*old)+1)
	for k, v := range *old {
		newMap[k] = v
	}
	newMap[key] = value
	// Store the new map atomically
	c.valid.Store(&newMap)
}

// loadFromFile loads cache entries from the persistent file.
func (c *Cache) loadFromFile() error {
	f, err := os.Open(c.file)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with empty map
			m := make(map[string]string)
			c.valid.Store(&m)
			return nil
		}
		return err
	}
	defer f.Close()

	m := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// Format: ip hostname
		ip, hostname := splitTwo(line)
		if ip == "" || hostname == "" {
			continue
		}
		m[ip] = hostname
	}

	c.valid.Store(&m)
	return scanner.Err()
}

// Persist writes all entries to the persistent file.
func (c *Cache) Persist() error {
	m := c.valid.Load()

	f, err := os.Create(c.file)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for ip, hostname := range *m {
		if _, err := fmt.Fprintf(w, "%s %s\n", ip, hostname); err != nil {
			return err
		}
	}
	return w.Flush()
}

// Prune removes entries from the cache that are no longer valid.
func (c *Cache) Prune(domains []string) {
	old := c.valid.Load()
	newMap := make(map[string]string, len(*old))

	for ip, hostname := range *old {
		if matchDomain(hostname, domains) {
			newMap[ip] = hostname
		}
	}

	c.valid.Store(&newMap)
}

// Size returns the number of entries in the cache.
func (c *Cache) Size() int {
	m := c.valid.Load()
	return len(*m)
}

// Close is a no-op. Cache persistence is handled by Persist().
func (c *Cache) Close() error {
	return nil
}
