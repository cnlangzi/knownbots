// Package asn provides ASN (Autonomous System Number) verification for bots.
package asn

import (
	"net/netip"

	"github.com/kentik/patricia"
	"github.com/kentik/patricia/uint_tree"
)

// ASN provides fast O(1) IP lookup using Radix Trees for ASN prefix matching.
type ASN struct {
	asns []int
	v4   *uint_tree.TreeV4
	v6   *uint_tree.TreeV6
}

// NewASN creates a new ASN cache with empty trees.
func NewASN() *ASN {
	return &ASN{
		asns: []int{},
		v4:   uint_tree.NewTreeV4(),
		v6:   uint_tree.NewTreeV6(),
	}
}

// ASNs returns the list of ASN numbers in this cache.
func (c *ASN) ASNs() []int {
	return c.asns
}

// Add adds all prefixes from an ASN to the cache.
func (c *ASN) Add(asn int, prefixes []netip.Prefix) error {
	c.asns = append(c.asns, asn)
	for _, prefix := range prefixes {
		addr := prefix.Addr()
		if addr.Is4() {
			patriciaAddr, _, _ := patricia.ParseFromNetIPPrefix(prefix)
			if patriciaAddr != nil {
				c.v4.Add(*patriciaAddr, 1, nil)
			}
		} else {
			_, patriciaAddr, _ := patricia.ParseFromNetIPPrefix(prefix)
			if patriciaAddr != nil {
				c.v6.Add(*patriciaAddr, 1, nil)
			}
		}
	}
	return nil
}

// Contains checks if an IP exists in any of the loaded prefixes.
func (c *ASN) Contains(ip netip.Addr) bool {
	patriciaAddrV4, patriciaAddrV6, _ := patricia.ParseFromNetIPAddr(ip)
	if ip.Is4() && patriciaAddrV4 != nil {
		found, _ := c.v4.FindDeepestTag(*patriciaAddrV4)
		return found
	}
	if patriciaAddrV6 != nil {
		found, _ := c.v6.FindDeepestTag(*patriciaAddrV6)
		return found
	}
	return false
}

// Clone creates a copy of the ASN cache.
func (c *ASN) Clone() *ASN {
	asnsCopy := make([]int, len(c.asns))
	copy(asnsCopy, c.asns)
	return &ASN{
		asns: asnsCopy,
		v4:   c.v4.Clone(),
		v6:   c.v6.Clone(),
	}
}

// Count returns the total number of prefixes in the cache.
func (c *ASN) Count() int {
	return c.v4.CountTags() + c.v6.CountTags()
}

// IPv4Tree returns the IPv4 tree for testing.
func (c *ASN) IPv4Tree() *uint_tree.TreeV4 {
	return c.v4
}

// IPv6Tree returns the IPv6 tree for testing.
func (c *ASN) IPv6Tree() *uint_tree.TreeV6 {
	return c.v6
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

// ValidatePrefix checks if a prefix is valid for ASN use.
func ValidatePrefix(prefix netip.Prefix) bool {
	addr := prefix.Addr()
	if addr.IsLoopback() || addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	if prefix.Bits() == 0 {
		return false
	}
	return true
}

// FilterInvalidPrefixes removes invalid prefixes from a slice.
func FilterInvalidPrefixes(prefixes []netip.Prefix) []netip.Prefix {
	result := make([]netip.Prefix, 0, len(prefixes))
	for _, p := range prefixes {
		if ValidatePrefix(p) {
			result = append(result, p)
		}
	}
	return result
}
