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
