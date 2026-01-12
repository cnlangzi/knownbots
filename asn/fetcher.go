// Package asn provides ASN (Autonomous System Number) verification for bots.
package asn

import "net/netip"

// Fetcher defines the interface for fetching ASN IP prefixes.
type Fetcher interface {
	Fetch(asn int) ([]netip.Prefix, error)
}
