// Package domainpolicy canonicalizes the trusted-external-domain whitelist
// (issue #1200): the admin-configurable list of hostnames whose normal-risk
// external links skip the "leaving the site" confirmation on Web and Flutter.
//
// Entries are stored as plain lower-cased hostnames — no scheme, path, port,
// wildcard or IP literal — so the browser-side matcher can compare them with
// URL.hostname verbatim. An entry also covers its subdomains
// ("bilibili.com" matches "live.bilibili.com"); the reverse never holds.
// Entries that are themselves a public suffix ("com", "co.uk", "github.io")
// are rejected: they would exempt every tenant / every domain below them.
package domainpolicy

import (
	"net/netip"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

// MaxEntries caps the stored whitelist. Entries beyond the cap are reported
// as dropped so the save path rejects the submission instead of truncating.
const MaxEntries = 200

// maxDomainLength is the DNS name limit (RFC 1035 §3.1) including labels.
const maxDomainLength = 253

// maxLabelLength is the DNS label limit (RFC 1035 §3.1).
const maxLabelLength = 63

// CanonicalizeList trims, lower-cases, de-duplicates and validates every
// entry while preserving submission order. Blank entries are skipped silently
// (paste noise) and duplicates collapse onto the first occurrence; invalid
// entries and entries beyond MaxEntries are returned in dropped so callers can
// reject the whole submission (save path) or filter stored values (read path).
func CanonicalizeList(entries []string) (kept []string, dropped []string) {
	kept = make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, raw := range entries {
		value, ok := Canonicalize(raw)
		if !ok {
			dropped = append(dropped, raw)
			continue
		}
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		if len(kept) >= MaxEntries {
			dropped = append(dropped, raw)
			continue
		}
		seen[value] = struct{}{}
		kept = append(kept, value)
	}
	return kept, dropped
}

// Canonicalize validates one whitelist entry and returns its stored form.
// Empty entries are valid and yield "". The stored form is ASCII lower-case
// with a trailing root dot removed; IDN input is converted to its punycode
// form so it compares equal to the browser's URL.hostname.
func Canonicalize(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	if len(value) > maxDomainLength {
		return value, false
	}
	lowered := strings.ToLower(strings.TrimSuffix(value, "."))
	if lowered == "" {
		return value, false
	}
	// Reject anything outside the hostname alphabet before IDNA mapping so
	// "github.com/path", "github.com:443", "*.github.com" and control
	// characters fail here instead of relying on the mapper's leniency.
	for _, r := range lowered {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '.', r > 0x7f:
		default:
			return value, false
		}
	}
	ascii, err := idna.Lookup.ToASCII(lowered)
	if err != nil {
		return value, false
	}
	ascii = strings.ToLower(ascii)
	if len(ascii) > maxDomainLength {
		return value, false
	}
	// A single label (localhost, an intranet short name) is not a public site.
	if !strings.Contains(ascii, ".") {
		return value, false
	}
	// IP literals are not domains: the whitelist targets public sites, and the
	// client-side registrable-domain logic cannot reason about addresses.
	if _, err := netip.ParseAddr(ascii); err == nil {
		return value, false
	}
	for _, label := range strings.Split(ascii, ".") {
		if label == "" || len(label) > maxLabelLength || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return value, false
		}
	}
	// An entry that is a public suffix would exempt every domain under it
	// (e.g. "github.io" exempts all GitHub Pages tenants, "com" everything).
	if suffix, _ := publicsuffix.PublicSuffix(ascii); suffix == ascii {
		return value, false
	}
	return ascii, true
}
