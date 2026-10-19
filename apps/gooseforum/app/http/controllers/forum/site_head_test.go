package forum

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestSiteHeadHTMLStripsDocumentAndOwnedMetadata(t *testing.T) {
	input := `<head><meta charset="utf-8"><meta name="viewport" content="wrong"><title>Wrong</title>
<meta name="description" content="wrong"><meta name="robots" content="index"><meta property="og:title" content="wrong">
<link rel="canonical" href="/wrong"><link rel="preconnect" href="https://cdn.example.test"><link rel="stylesheet" href="https://cdn.example.test/site.css"><link rel="preload" href="https://cdn.example.test/site.js" as="script">
<script defer src="/extra.js"></script><meta name="google-site-verification" content="ok"></head>`
	output := string(siteHeadHTML(input))
	for _, prohibited := range []string{"<head", "</head", "<title", "charset", "viewport", "description", "robots", "og:title", "canonical"} {
		if strings.Contains(strings.ToLower(output), prohibited) {
			t.Errorf("injected metadata %q survived: %s", prohibited, output)
		}
	}
	for _, required := range []string{`rel="preconnect"`, `rel="stylesheet"`, `rel="preload"`, "/extra.js", "google-site-verification"} {
		if !strings.Contains(output, required) {
			t.Errorf("custom resource %q was dropped: %s", required, output)
		}
	}
}

func TestSiteHeadHTMLDoesNotEmitUnclosedScript(t *testing.T) {
	if got := siteHeadHTML(`<script src="/bad.js">`); got != "" {
		t.Fatalf("incomplete script emitted: %s", got)
	}
	if got := siteHeadHTML(`<link rel="preconnect" href="https://cdn.example.test"><style>.foo{color:red}`); got != `<link rel="preconnect" href="https://cdn.example.test">` {
		t.Fatalf("incomplete style or its contents emitted: %s", got)
	}
	if got := siteHeadHTML(`<style>.foo{color:red}</style>`); !strings.Contains(string(got), ".foo{color:red}") {
		t.Fatalf("valid custom style was lost: %s", got)
	}
}

func TestSiteHeadHTMLIgnoresBodyResourcesFromLegacyDocument(t *testing.T) {
	input := `<html><head><meta name="google-site-verification" content="head"><link rel="preconnect" href="https://head.example.test"></head><body><meta name="google-site-verification" content="body"><link rel="preconnect" href="https://body.example.test"><script src="/body.js"></script><style>.body{color:red}</style></body></html>`
	output := string(siteHeadHTML(input))
	if !strings.Contains(output, `content="head"`) || !strings.Contains(output, `href="https://head.example.test"`) {
		t.Fatalf("valid head resources were lost: %s", output)
	}
	for _, bodyOnly := range []string{`content="body"`, "https://body.example.test", "/body.js", ".body{color:red}"} {
		if strings.Contains(output, bodyOnly) {
			t.Errorf("body resource %q was moved into head: %s", bodyOnly, output)
		}
	}
}

func TestSiteHeadHTMLDoesNotLeakTagsFromMalformedSelfClosingScript(t *testing.T) {
	output := string(siteHeadHTML(`<script src="/extra.js"/><meta name="google-site-verification" content="script-body"></script><link rel="preconnect" href="https://cdn.example.test">`))
	doc, err := html.Parse(strings.NewReader("<!doctype html><html><head>" + output + "</head><body></body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	scriptCount, verificationMetaCount, preconnectCount := 0, 0, 0
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "script":
				scriptCount++
			case "meta":
				for _, attr := range node.Attr {
					if attr.Key == "name" && attr.Val == "google-site-verification" {
						verificationMetaCount++
					}
				}
			case "link":
				for _, attr := range node.Attr {
					if attr.Key == "rel" && attr.Val == "preconnect" {
						preconnectCount++
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)
	if scriptCount != 0 || verificationMetaCount != 0 || preconnectCount != 1 {
		t.Fatalf("malformed self-closing script leaked or swallowed later tags: scripts=%d verification metas=%d preconnects=%d output=%s", scriptCount, verificationMetaCount, preconnectCount, output)
	}
}

func TestSSRShellKeepsSinglePageMetadataWithLegacyCustomHead(t *testing.T) {
	payload := PagePayload{
		Meta: PageMeta{
			Title: "Page title", Description: "Summary", Robots: "noindex",
			Canonical: "/page", PrevURL: "/previous", NextURL: "/next",
			OpenGraph: &OpenGraphMeta{Title: "Page title", Description: "Summary", PublishedTime: "2026-10-19"},
			Twitter:   &TwitterMeta{Card: "summary", Title: "Page title", Description: "Summary"},
		},
		Layout: LayoutPayload{Site: SitePayload{ExternalLinks: `<html><head><meta charset="utf-8"><meta http-equiv="refresh" content="0"><title>Wrong</title><meta name="viewport" content="bad"><meta name="description" content="bad"><meta name="robots" content="index"><meta name="author" content="YourTJ Community"><meta name="keywords" content="forum,community"><meta name="theme-color" content="red"><meta name="color-scheme" content="dark"><meta property="og:title" content="wrong"><meta property="article:published_time" content="wrong"><meta name="twitter:title" content="wrong"><link rel="canonical" href="/wrong"><link rel="prev" href="/wrong"><link rel="next" href="/wrong"><link rel="manifest" href="/wrong"><link rel="icon" href="/wrong"><link rel="shortcut icon" href="/wrong"><link rel="apple-touch-icon" href="/wrong"><link rel="apple-touch-icon-precomposed" href="/wrong"><link rel="preconnect" href="https://cdn.example.test"></head><body><meta name="google-site-verification" content="from-body"></body></html>`}},
	}
	var buf strings.Builder
	if err := currentRegistry.render(&buf, "app_shell.gohtml", templateData{Payload: payload, Lang: "en"}); err != nil {
		t.Fatal(err)
	}
	page := buf.String()
	for _, tag := range []string{"<html", "<head>", "<body", `<meta name="viewport"`, "<title>", `<meta name="description"`, `<meta name="robots"`, `<meta name="theme-color"`, `<meta name="color-scheme"`, `<meta property="og:title"`, `<meta property="article:published_time"`, `<meta name="twitter:title"`, `<link rel="canonical"`, `<link rel="prev"`, `<link rel="next"`, `<link rel="manifest"`, `<link rel="apple-touch-icon"`} {
		if n := strings.Count(page, tag); n != 1 {
			t.Errorf("SSR has %d copies of %q, want 1", n, tag)
		}
	}
	for _, tag := range []string{`<meta name="author"`, `<meta name="keywords"`} {
		if n := strings.Count(page, tag); n != 1 {
			t.Errorf("SSR has %d copies of custom metadata %q, want 1", n, tag)
		}
	}
	for _, tag := range []string{`<link rel="shortcut icon"`, `<link rel="apple-touch-icon-precomposed"`} {
		if strings.Contains(page, tag) {
			t.Errorf("system-owned link %q survived custom filtering", tag)
		}
	}
	if strings.Contains(page, "<title>Wrong</title>") || strings.Contains(page, `content="from-body"`) || !strings.Contains(page, `rel="preconnect"`) {
		t.Fatal("custom head filtered incorrectly")
	}
}

func TestSSRShellUsesSiteDescriptionWhenPageDescriptionIsEmpty(t *testing.T) {
	payload := PagePayload{
		Meta:   PageMeta{Title: "Search"},
		Layout: LayoutPayload{Site: SitePayload{Description: "Forum summary"}},
	}
	var buf strings.Builder
	if err := currentRegistry.render(&buf, "app_shell.gohtml", templateData{Payload: payload, Lang: "en"}); err != nil {
		t.Fatal(err)
	}
	page := buf.String()
	if n := strings.Count(page, `<meta name="description"`); n != 1 {
		t.Fatalf("SSR has %d descriptions, want one site fallback", n)
	}
	if !strings.Contains(page, `<meta name="description" content="Forum summary">`) {
		t.Fatalf("site description was not used as the fallback: %s", page)
	}
}

func TestSSRShellUsesDefaultRobotsWhenPageDirectiveIsEmpty(t *testing.T) {
	payload := PagePayload{Meta: PageMeta{Title: "Home"}}
	var buf strings.Builder
	if err := currentRegistry.render(&buf, "app_shell.gohtml", templateData{Payload: payload, Lang: "en"}); err != nil {
		t.Fatal(err)
	}
	page := buf.String()
	if n := strings.Count(page, `<meta name="robots"`); n != 1 {
		t.Fatalf("SSR has %d robots directives, want one default", n)
	}
	if !strings.Contains(page, `<meta name="robots" content="index, follow">`) {
		t.Fatalf("default robots directive missing: %s", page)
	}
}
