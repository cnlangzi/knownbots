// Package knownbots provides bot verification through UserAgent and IP validation.
package knownbots

import (
	"strings"
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
