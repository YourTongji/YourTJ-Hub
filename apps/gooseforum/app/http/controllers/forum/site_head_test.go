package forum

import (
	"strings"
	"testing"
)

func TestSiteHeadHTMLStripsDocumentAndOwnedMetadata(t *testing.T) {
	input := `<head><meta charset="utf-8"><meta name="viewport" content="wrong"><title>Wrong</title>
<meta name="description" content="wrong"><meta name="robots" content="index"><meta property="og:title" content="wrong">
<link rel="canonical" href="/wrong"><link rel="preconnect" href="https://cdn.example.test">
<script defer src="/extra.js"></script><meta name="google-site-verification" content="ok"></head>`
	output := string(siteHeadHTML(input))
	for _, prohibited := range []string{"<head", "</head", "<title", "charset", "viewport", "description", "robots", "og:title", "canonical"} {
		if strings.Contains(strings.ToLower(output), prohibited) {
			t.Errorf("injected metadata %q survived: %s", prohibited, output)
		}
	}
	for _, required := range []string{`rel="preconnect"`, "/extra.js", "google-site-verification"} {
		if !strings.Contains(output, required) {
			t.Errorf("custom resource %q was dropped: %s", required, output)
		}
	}
}

func TestSiteHeadHTMLDoesNotEmitUnclosedScript(t *testing.T) {
	if got := siteHeadHTML(`<script src="/bad.js">`); got != "" {
		t.Fatalf("incomplete script emitted: %s", got)
	}
	if got := siteHeadHTML(`<style>.foo{color:red}</style>`); !strings.Contains(string(got), ".foo{color:red}") {
		t.Fatalf("valid custom style was lost: %s", got)
	}
}

func TestSSRShellKeepsSinglePageMetadataWithLegacyCustomHead(t *testing.T) {
	payload := PagePayload{
		Meta: PageMeta{Title: "Page title", Description: "Summary", Robots: "noindex"},
		Layout: LayoutPayload{Site: SitePayload{ExternalLinks: `<head><title>Wrong</title><meta name="viewport" content="bad"><meta name="description" content="bad"><link rel="preconnect" href="https://cdn.example.test"></head>`}},
	}
	var buf strings.Builder
	if err := currentRegistry.render(&buf, "app_shell.gohtml", templateData{Payload: payload, Lang: "en"}); err != nil {
		t.Fatal(err)
	}
	page := buf.String()
	for _, tag := range []string{"<head>", `<meta name="viewport"`, "<title>", `<meta name="description"`, `<meta name="robots"`} {
		if n := strings.Count(page, tag); n != 1 {
			t.Errorf("SSR has %d copies of %q, want 1", n, tag)
		}
	}
	if strings.Contains(page, "<title>Wrong</title>") || !strings.Contains(page, `rel="preconnect"`) {
		t.Fatal("custom head filtered incorrectly")
	}
}
