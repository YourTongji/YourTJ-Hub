// Package headmarkup reduces the administrator-configured site setting
// "external links / meta tags" to markup that may live inside the page's own
// <head> (issue #1244).
//
// The field was rendered verbatim into the shell's <head>, so an administrator
// pasting a whole <head>…</head> block produced a nested <head> plus duplicate
// viewport/title/description tags. Sanitize keeps the field's real content
// (custom meta/link/script/style/noscript) and strips only document structure
// and the document metadata the shell emits per page: <title>, the charset
// (including http-equiv=content-type), the viewport/description/robots/
// color-scheme/theme-color metas, og:/twitter:/article:-prefixed metas, and the
// canonical/prev/next/manifest/icon/apple-touch-icon links (matched by rel
// token).
//
// This is a structural boundary, not a trust boundary: the field is trusted
// administrator content, so no URL or script whitelist is applied here.
package headmarkup

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Sanitize returns the subset of raw that is valid inside the shell's <head>:
// the parser consumes <!DOCTYPE>/<html>/<head>/<body> and their children are
// reached through the head element, so body subtrees and loose text are
// dropped, while the document metadata the shell renders per page (<title>, the
// charset/viewport/description/robots/color-scheme/theme-color metas, og:/
// twitter:/article:-prefixed metas and the canonical/prev/next/manifest/icon
// links) is removed to keep those tags unique per page.
// Sanitize is idempotent; empty or whitespace-only input yields "".
func Sanitize(raw string) string {
	// html.Parse fails on a tokenizer error and on its internal panic for input
	// nested deeper than 512 elements; keep the fail-closed result (no markup).
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return ""
	}
	// The parser synthesizes <html><head>…</head> for any input, so a missing
	// head means the tree is unusable; keep the fail-closed result.
	head := findHead(doc)
	if head == nil {
		return ""
	}
	kept := make([]string, 0, 4)
	for child := head.FirstChild; child != nil; child = child.NextSibling {
		// Whitespace text nodes and comments carry no head markup; stray text
		// that reached head is dropped for the same reason.
		if child.Type != html.ElementNode || shellOwned(child) {
			continue
		}
		var rendered bytes.Buffer
		if err := html.Render(&rendered, child); err != nil {
			continue
		}
		kept = append(kept, rendered.String())
	}
	return strings.Join(kept, "\n")
}

func findHead(node *html.Node) *html.Node {
	if node.Type == html.ElementNode && node.DataAtom == atom.Head {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if head := findHead(child); head != nil {
			return head
		}
	}
	return nil
}

// shellOwned reports the head children the shell (resource/templates/layout/
// app.gohtml) already emits. Keeping a configured copy would duplicate document
// metadata the page sets per request.
func shellOwned(node *html.Node) bool {
	switch node.DataAtom {
	case atom.Html, atom.Head, atom.Body:
		// Not reachable from a parsed head child, but nested document structure is
		// exactly the defect this package exists to prevent: keep the guard.
		return true
	case atom.Title:
		return true
	case atom.Meta:
		return shellOwnedMeta(node)
	case atom.Link:
		return shellOwnedLink(node)
	}
	return false
}

// shellOwnedMeta matches the shell's charset/viewport/description/robots/
// color-scheme/theme-color metas and the og:/twitter:/article:-prefixed entries
// it emits for a page. The parser lower-cases attribute names but not values.
// An http-equiv=content-type declaration duplicates the shell's <meta charset>,
// but other policies (CSP, refresh, …) pass through.
func shellOwnedMeta(node *html.Node) bool {
	for _, attr := range node.Attr {
		switch attr.Key {
		case "charset":
			return true
		case "http-equiv":
			if strings.EqualFold(strings.TrimSpace(attr.Val), "content-type") {
				return true
			}
		case "name", "property":
			name := strings.ToLower(strings.TrimSpace(attr.Val))
			switch name {
			case "viewport", "description", "robots", "color-scheme", "theme-color":
				return true
			}
			if strings.HasPrefix(name, "og:") || strings.HasPrefix(name, "twitter:") ||
				strings.HasPrefix(name, "article:") {
				return true
			}
		}
	}
	return false
}

// shellOwnedLink matches the shell's canonical/prev/next/manifest/icon links;
// other link relations (preconnect, stylesheet, alternate, …) pass through.
// rel is a token set, so any matching token makes the whole link shell-owned
// (rel="canonical alternate", rel="shortcut icon").
func shellOwnedLink(node *html.Node) bool {
	for _, attr := range node.Attr {
		if attr.Key != "rel" {
			continue
		}
		for _, token := range strings.Fields(strings.ToLower(attr.Val)) {
			switch token {
			case "canonical", "prev", "next", "manifest", "icon", "apple-touch-icon":
				return true
			}
		}
	}
	return false
}
