// Package headmarkup reduces the administrator-configured site setting
// "external links / meta tags" to markup that may live inside the page's own
// <head> (issue #1244).
//
// The field was rendered verbatim into the shell's <head>, so an administrator
// pasting a whole <head>…</head> block produced a nested <head> plus duplicate
// viewport/title/description tags. Sanitize keeps the field's real content
// (custom meta/link/script/style/noscript) and strips only document structure
// and the tags the shell emits itself.
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
// dropped, while <title> and the charset/viewport/description/robots metas the
// shell renders itself are removed to keep those tags unique per page.
// Sanitize is idempotent; empty or whitespace-only input yields "".
func Sanitize(raw string) string {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return ""
	}
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

// shellOwned reports the head children the templates already emit. Keeping a
// configured copy would duplicate metadata the page sets per request.
func shellOwned(node *html.Node) bool {
	switch node.DataAtom {
	case atom.Html, atom.Head, atom.Body, atom.Title:
		return true
	case atom.Meta:
		return shellOwnedMeta(node)
	}
	return false
}

// shellOwnedMeta matches the shell's charset/viewport/description/robots metas.
// The parser lower-cases attribute names but not values. An http-equiv=content-type
// declaration duplicates the shell's <meta charset>, but other policies (CSP,
// refresh, …) pass through.
func shellOwnedMeta(node *html.Node) bool {
	for _, attr := range node.Attr {
		switch attr.Key {
		case "charset":
			return true
		case "http-equiv":
			if strings.EqualFold(strings.TrimSpace(attr.Val), "content-type") {
				return true
			}
		case "name":
			switch strings.ToLower(strings.TrimSpace(attr.Val)) {
			case "viewport", "description", "robots":
				return true
			}
		}
	}
	return false
}
