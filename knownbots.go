// Package knownbots provides bot verification through UserAgent and IP validation.
package knownbots

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnlangzi/knownbots/parser"
)

// matchDomain checks if the hostname matches any of the given domains.
func matchDomain(hostname string, domains []string) bool {
	for _, domain := range domains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			return true
		}
	}
	return false
}

// containsWord checks if word appears in text as a standalone word.
func containsWord(text, word string) bool {
	if word == "" {
		return false
	}

	idx := 0
	for idx < len(text) {
		i := strings.Index(text[idx:], word)
		if i == -1 {
			break
		}

		pos := idx + i
		beforeOK := pos == 0 || !isAlphaNumeric(text[pos-1])
		afterOK := pos+len(word) == len(text) || !isAlphaNumeric(text[pos+len(word)])

		if beforeOK && afterOK {
			return true
		}

		idx = pos + 1
	}

	return false
}

// isAlphaNumeric checks if a byte is alphanumeric.
func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// splitTwo splits a line into two parts at the first space.
// Returns empty strings if the line doesn't contain exactly two parts.
func splitTwo(s string) (string, string) {
	for i, c := range s {
		if c == ' ' {
			if i <= 0 || i >= len(s)-1 {
				return "", ""
			}
			return s[:i], s[i+1:]
		}
	}
	return "", ""
}

// downloadIPs fetches IP ranges from URLs and parses using the bot's registered parser.
func downloadIPs(httpClient *http.Client, bot *Bot) []netip.Prefix {
	var allPrefixes []netip.Prefix
	p := parser.Get(bot.Parser)

	for _, url := range bot.URLs {
		prefixes := fetchAndParse(httpClient, p, url)
		allPrefixes = append(allPrefixes, prefixes...)
	}
	return allPrefixes
}

// fetchAndParse fetches IP ranges from a single URL and parses them.
func fetchAndParse(httpClient *http.Client, p parser.Parser, url string) []netip.Prefix {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	prefixes, err := p.Parse(resp.Body)
	if err != nil {
		log.Printf("[knownbots] failed to parse IPs from %s: %v", url, err)
		return nil
	}
	return prefixes
}

// writeIPs persists IP ranges to file (failure is OK).
func writeIPs(path string, prefixes []netip.Prefix) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, prefix := range prefixes {
		if _, err := fmt.Fprintln(w, prefix.String()); err != nil {
			return
		}
	}
	w.Flush()
}

// buildUAIndex creates a byte-level index mapping first characters to bot candidates.
func buildUAIndex(bots []*Bot) map[byte][]*Bot {
	index := make(map[byte][]*Bot, 26)

	for _, bot := range bots {
		if bot.UA == "" {
			continue
		}

		firstChar := bot.UA[0]
		index[firstChar] = append(index[firstChar], bot)
	}

	return index
}
