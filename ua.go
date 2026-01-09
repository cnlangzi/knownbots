package knownbots

import (
	"strings"
)

// BrowserKind represents the classification of a UserAgent structure.
type BrowserKind string

const (
	Browser    BrowserKind = "browser"    // Legitimate browser
	Suspicious BrowserKind = "suspicious" // Claims to be browser but malformed
	Unknown    BrowserKind = "unknown"    // Not browser-like
)

// Browser product names - specific browser identifiers in UA strings.
// Checked in addition to rendering engines for comprehensive coverage.
var browserProducts = []string{
	// Global mainstream
	"Chrome/", "CriOS/", "Safari/", "Mobile Safari",
	"Firefox/", "FxiOS/", "Edg/", "EdgA/", "EdgiOS/",
	"OPR/", "Opera/", "Brave/", "Vivaldi/", "Arc/",
	// China
	"UCBrowser/", "UCWEB/", "360SE/", "QIHU/", "QQBrowser/", "QQ/",
	"baidubrowser/", "Baidu/", "MetaSr/", "SogouExplorer",
	"Maxthon/", "mxBrowser/", "LBBROWSER/", "Liebao",
	"TaoBrowser/", "MiuiBrowser/", "XiaoMi/", "Mi/",
	"HuaweiBrowser/", "flyflow/", "QQDownload/",
	// Korea
	"Whale/",
	// Japan
	"Sleipnir/",
	// Russia
	"YaBrowser/", "Yandex/",
	// Vietnam
	"CocCoc/",
	// India
	"JioSphere/", "JioPages/",
	// Privacy/security
	"Tor Browser/", "Waterfox/", "Pale Moon/", "Basilisk/", "K-Meleon/",
	"Comodo_Dragon/", "Comodo Dragon", "Iron/", "LibreWolf/",
	"Iceweasel/", "IceCat/", "SeaMonkey/", "Epiphany/",
	"Midori/", "Konqueror/", "Silk/",
}

// Rendering engines - core browser engine identifiers.
// Keep as strong signals (typically appear with a version like "AppleWebKit/537.36").
var browserEngines = []string{
	"AppleWebKit/", // WebKit family (Chrome, Safari, Edge, Brave, Vivaldi, all iOS browsers)
	"Gecko/",       // Firefox family (e.g. Gecko/20100101)
	"Trident/",     // Internet Explorer (legacy)
	"Presto/",      // Opera (pre-Chromium, legacy)
	"Goanna/",      // Pale Moon family (Gecko fork)
}

func uaHasControlChars(ua string) bool {
	for i := 0; i < len(ua); i++ {
		b := ua[i]
		if b < 0x20 || b == 0x7f {
			return true
		}
	}
	return false
}

func uaParenInfo(ua string) (hasPair bool, balanceOK bool) {
	bal := 0
	seenLeft := false
	escaped := false
	for i := 0; i < len(ua); i++ {
		if escaped {
			escaped = false
			continue
		}
		switch ua[i] {
		case '\\':
			escaped = true
		case '(':
			bal++
			seenLeft = true
		case ')':
			bal--
			if bal < 0 {
				return false, false
			}
		}
	}
	if !seenLeft {
		return false, true
	}
	// hasPair: at least one '(' and a matching ')' somewhere (balance ends at 0)
	return true, bal == 0 && !escaped
}

func uaContainsAny(ua string, tokens []string) bool {
	for _, token := range tokens {
		if strings.Contains(ua, token) {
			return true
		}
	}
	return false
}

func classifyUA(ua string) BrowserKind {
	if ua == "" {
		return Unknown
	}

	hasPair, parenBalanceOK := uaParenInfo(ua)
	hasValidParens := hasPair && parenBalanceOK
	claimsMozilla := strings.HasPrefix(ua, "Mozilla")

	if claimsMozilla {
		if uaHasControlChars(ua) {
			return Suspicious
		}
		if !parenBalanceOK {
			return Suspicious
		}
		if len(ua) < 16 {
			return Suspicious
		}
		if !hasPair {
			return Suspicious
		}
	}

	hasBrowserProduct := uaContainsAny(ua, browserProducts)
	hasEngine := uaContainsAny(ua, browserEngines)

	if hasBrowserProduct || hasEngine {
		if !hasValidParens {
			return Suspicious
		}
		return Browser
	}

	if claimsMozilla && hasValidParens {
		return Suspicious
	}

	return Unknown
}
