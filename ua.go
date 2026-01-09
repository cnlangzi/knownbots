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

// Rendering engines - core browser engine identifiers.
// Presence with valid UA structure indicates a browser.
var browserEngines = []string{
	"AppleWebKit", // WebKit (Chrome, Safari, Edge, Brave, Vivaldi, all iOS browsers)
	"Gecko",       // Firefox and forks (Firefox, SeaMonkey, Pale Moon, etc.)
	"Trident",     // Internet Explorer (legacy)
	"Presto",      // Opera (pre-Chromium, legacy)
	"Goanna",      // Pale Moon family (Gecko fork)
}

// Browser product names - specific browser identifiers in UA strings.
// Checked in addition to rendering engines for comprehensive coverage.
var browserProducts = []string{
	"Mozilla", // Core browser prefix (standalone catches partial UAs)
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
	// Mobile platforms
	"Android", "Linux; U; Android",
}

// classifyUA categorizes a UserAgent into browser, suspicious, or unknown.
// Uses all known browser patterns for comprehensive detection.
func classifyUA(ua string) BrowserKind {
	if ua == "" {
		return Unknown
	}

	// Check valid parentheses structure
	parenStart := strings.Index(ua, "(")
	parenEnd := strings.Index(ua, ")")
	hasValidParens := parenStart != -1 && parenEnd != -1 && parenEnd > parenStart+1

	// Check browser product presence
	hasBrowserProduct := false
	for _, product := range browserProducts {
		if strings.Contains(ua, product) {
			hasBrowserProduct = true
			break
		}
	}

	// Check rendering engine presence
	hasEngine := false
	for _, engine := range browserEngines {
		if strings.Contains(ua, engine) {
			hasEngine = true
			break
		}
	}

	// Check Mozilla prefix (most browsers start with this)
	hasMozillaPrefix := strings.HasPrefix(ua, "Mozilla/")

	// Decision tree ordered by commonality and specificity
	if hasBrowserProduct && hasValidParens {
		return Browser
	}
	if hasEngine && hasValidParens {
		return Browser
	}
	if hasMozillaPrefix && hasValidParens {
		return Browser
	}
	if hasBrowserProduct {
		return Suspicious
	}
	if hasEngine {
		return Suspicious
	}
	if hasMozillaPrefix {
		return Suspicious
	}
	return Unknown
}
