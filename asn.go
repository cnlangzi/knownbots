package knownbots

import (
	"net/netip"

	"github.com/cnlangzi/knownbots/asn"
)

// ASN provides fast IP lookup for ASN prefix matching using IPTree.
// Immutable after creation - use atomic.Pointer[ASN] at call site for thread-safety.
type ASN struct {
	asns []int
	tree *IPTree
}

// NewASN creates a new ASN cache with an empty tree.
func NewASN() *ASN {
	return &ASN{
		asns: []int{},
		tree: NewIPTree(),
	}
}

// ASNs returns the list of ASN numbers in this cache.
func (c *ASN) ASNs() []int {
	return c.asns
}

// Add adds all prefixes from an ASN to the cache.
func (c *ASN) Add(asn int, prefixes []netip.Prefix) {
	seen := false
	for _, existing := range c.asns {
		if existing == asn {
			seen = true
			break
		}
	}
	if !seen {
		c.asns = append(c.asns, asn)
	}

	for _, prefix := range prefixes {
		c.tree.Add(prefix)
	}
}

// Contains checks if an IP exists in any of the loaded prefixes.
func (c *ASN) Contains(ip netip.Addr) bool {
	return c.tree.Contains(ip)
}

// Count returns the total number of prefixes in the cache.
func (c *ASN) Count() int {
	return c.tree.Count()
}

// Deduplicate removes duplicate prefixes from a slice.
func Deduplicate(prefixes []netip.Prefix) []netip.Prefix {
	seen := make(map[string]bool)
	result := make([]netip.Prefix, 0, len(prefixes))
	for _, p := range prefixes {
		key := p.String()
		if !seen[key] {
			seen[key] = true
			result = append(result, p)
		}
	}
	return result
}

// FilterInvalidPrefixes removes invalid prefixes from a slice.
func FilterInvalidPrefixes(prefixes []netip.Prefix) []netip.Prefix {
	result := make([]netip.Prefix, 0, len(prefixes))
	for _, p := range prefixes {
		if asn.ValidatePrefix(p) {
			result = append(result, p)
		}
	}
	return result
}
