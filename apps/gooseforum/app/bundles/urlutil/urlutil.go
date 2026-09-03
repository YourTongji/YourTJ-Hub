// Package urlutil validates admin-configurable URLs before they are stored
// (issue #409). Admin fields split into per-field policies (site navigation,
// external jumps, images, contact actions); every policy runs the same
// canonicalization pipeline first:
//
//  1. Trim surrounding whitespace.
//  2. HTML-entity decode, then reject control characters (browsers strip tab
//     and newline before parsing, so "java\tscript:..." or its entity-encoded
//     form "java&#x09;script:..." would otherwise become a javascript: scheme).
//  3. Lower-case scheme comparison against the policy whitelist.
//
// Protocol-relative URLs ("//host/path") are rejected for every kind, and
// http(s) URLs must carry a host. Empty values are always valid (optional
// admin fields). Values are stored trimmed but otherwise verbatim.
package urlutil

import (
	"html"
	"net/url"
	"strings"
)

// Kind is the security policy of an admin-configurable URL field.
type Kind int

const (
	// SiteLink covers in-site navigation and generic links (chrome nav items,
	// footer links): a relative site path or an absolute http(s) URL.
	SiteLink Kind = iota
	// External covers absolute http(s) only: sponsor links, friend links and
	// the site base URL.
	External
	// Image covers logo/avatar URLs: a site-relative path or http(s).
	Image
	// Contact covers action buttons (sponsors contact button): a relative
	// path, http(s), or a mailto: address (its built-in default).
	Contact
)

const maxURLLength = 2048

// Canonicalize trims and validates raw against kind. It returns the value a
// caller should persist and whether that value satisfies the policy.
func Canonicalize(kind Kind, raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	if len(value) > maxURLLength {
		return value, false
	}
	decoded := html.UnescapeString(value)
	if containsControl(decoded) || len(decoded) > maxURLLength {
		return value, false
	}
	scheme, host, opaque, protocolRelative := splitURL(decoded)
	if protocolRelative {
		return value, false
	}
	if scheme == "" {
		// Scheme-less values are site-relative paths. Only External requires
		// an absolute http(s) URL.
		return value, kind != External
	}
	switch kind {
	case External:
		return value, isAbsoluteHTTP(scheme, host)
	case SiteLink, Image:
		return value, isAbsoluteHTTP(scheme, host)
	case Contact:
		return value, isAbsoluteHTTP(scheme, host) || (scheme == "mailto" && opaque != "")
	default:
		return value, false
	}
}

// IsValid reports whether raw satisfies the kind policy.
func IsValid(kind Kind, raw string) bool {
	_, ok := Canonicalize(kind, raw)
	return ok
}

// Clean returns the canonical value for read/render paths, or "" when the
// stored value violates the kind policy (historical dirty configs degrade to
// an empty link instead of an executable href).
func Clean(kind Kind, raw string) string {
	value, ok := Canonicalize(kind, raw)
	if !ok {
		return ""
	}
	return value
}

func containsControl(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// splitURL returns the lower-cased scheme, host, opaque part and whether the
// value is a protocol-relative URL. Relative values yield an empty scheme.
func splitURL(value string) (scheme, host, opaque string, protocolRelative bool) {
	if strings.HasPrefix(value, "//") {
		return "", "", "", true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", "", "", false
	}
	return strings.ToLower(parsed.Scheme), parsed.Host, parsed.Opaque, false
}

func isAbsoluteHTTP(scheme, host string) bool {
	return (scheme == "http" || scheme == "https") && host != ""
}
