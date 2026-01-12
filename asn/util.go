package asn

import "net/netip"

// ValidatePrefix checks if a prefix is valid for ASN cache.
// Returns false for loopback, multicast, unspecified, or /0 prefixes.
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

// Distinct removes invalid and duplicate prefixes from a slice.
func Distinct(prefixes []netip.Prefix) []netip.Prefix {
	seen := make(map[string]bool)
	result := make([]netip.Prefix, 0, len(prefixes))
	for _, p := range prefixes {
		if !ValidatePrefix(p) {
			continue
		}
		key := p.String()
		if !seen[key] {
			seen[key] = true
			result = append(result, p)
		}
	}
	return result
}
