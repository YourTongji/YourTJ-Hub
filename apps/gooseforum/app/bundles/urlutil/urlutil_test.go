package urlutil

import "testing"

func TestCanonicalizeSiteLink(t *testing.T) {
	valid := []string{
		"",
		"/sponsors",
		"sponsors",
		"/wiki/a?b=c#frag",
		"https://example.com/path?q=1",
		"http://example.com",
		"https://example.com/站内?x=中文",
	}
	for _, raw := range valid {
		if got, ok := Canonicalize(SiteLink, raw); !ok {
			t.Errorf("SiteLink %q: want valid, got invalid", raw)
		} else if got != raw {
			t.Errorf("SiteLink %q: canonical = %q, want unchanged", raw, got)
		}
	}
	// Trimmed values are stored trimmed.
	if got, ok := Canonicalize(SiteLink, "  /sponsors  "); !ok || got != "/sponsors" {
		t.Errorf("SiteLink padded: got (%q,%v), want trimmed valid", got, ok)
	}
	invalid := []string{
		"javascript:alert(1)",
		"JaVaScRiPt:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=",
		"vbscript:msgbox(1)",
		"file:///etc/passwd",
		"//evil.example.com/path",
		"///evil.example.com/path",
		"java\nscript:alert(1)",
		"java\tscript:alert(1)",
		"java\rscript:alert(1)",
		"jav\u0000ascript:alert(1)",
		"javascript:alert(1)\n",
	}
	for _, raw := range invalid {
		if _, ok := Canonicalize(SiteLink, raw); ok {
			t.Errorf("SiteLink %q: want invalid, got valid", raw)
		}
	}
}

func TestCanonicalizeEncodedDisguises(t *testing.T) {
	// HTML entities are decoded before scheme parsing.
	invalid := []string{
		"jav&#x61;script:alert(1)",
		"jav&#97;script:alert(1)",
		"javascript&#58;alert(1)",
		"java&#x09;script:alert(1)",
		"&#106;&#97;&#118;&#97;&#115;&#99;&#114;&#105;&#112;&#116;&#58;alert(1)",
	}
	for _, raw := range invalid {
		if _, ok := Canonicalize(SiteLink, raw); ok {
			t.Errorf("SiteLink %q: want invalid, got valid", raw)
		}
	}
}

func TestCanonicalizeExternal(t *testing.T) {
	valid := []string{
		"",
		"https://example.com",
		"http://example.com/path",
	}
	for _, raw := range valid {
		if _, ok := Canonicalize(External, raw); !ok {
			t.Errorf("External %q: want valid, got invalid", raw)
		}
	}
	invalid := []string{
		"/sponsors",
		"sponsors",
		"//example.com",
		"mailto:a@example.com",
		"javascript:alert(1)",
		"data:image/png;base64,x",
		"https://",
		"http://",
		"ftp://example.com",
	}
	for _, raw := range invalid {
		if _, ok := Canonicalize(External, raw); ok {
			t.Errorf("External %q: want invalid, got valid", raw)
		}
	}
}

func TestCanonicalizeImage(t *testing.T) {
	valid := []string{
		"",
		"/static/pic/logo.webp",
		"https://cdn.example.com/a.png",
		"http://cdn.example.com/a.png",
	}
	for _, raw := range valid {
		if _, ok := Canonicalize(Image, raw); !ok {
			t.Errorf("Image %q: want valid, got invalid", raw)
		}
	}
	invalid := []string{
		"javascript:alert(1)",
		"//cdn.example.com/a.png",
		"file:///tmp/a.png",
		"data:image/svg+xml,<svg onload=alert(1)>",
	}
	for _, raw := range invalid {
		if _, ok := Canonicalize(Image, raw); ok {
			t.Errorf("Image %q: want invalid, got valid", raw)
		}
	}
}

func TestCanonicalizeContact(t *testing.T) {
	valid := []string{
		"",
		"/contact",
		"https://example.com/contact",
		"http://example.com/contact",
		"mailto:contact@example.com",
	}
	for _, raw := range valid {
		if _, ok := Canonicalize(Contact, raw); !ok {
			t.Errorf("Contact %q: want valid, got invalid", raw)
		}
	}
	invalid := []string{
		"mailto:",
		"javascript:alert(1)",
		"//example.com",
	}
	for _, raw := range invalid {
		if _, ok := Canonicalize(Contact, raw); ok {
			t.Errorf("Contact %q: want invalid, got valid", raw)
		}
	}
}

func TestCleanDegradesDirtyValues(t *testing.T) {
	if got := Clean(SiteLink, "javascript:alert(1)"); got != "" {
		t.Errorf("Clean(javascript:) = %q, want empty", got)
	}
	if got := Clean(External, "/sponsors"); got != "" {
		t.Errorf("Clean(relative external) = %q, want empty", got)
	}
	if got := Clean(SiteLink, "  /sponsors  "); got != "/sponsors" {
		t.Errorf("Clean(padded) = %q, want trimmed", got)
	}
	if got := Clean(SiteLink, ""); got != "" {
		t.Errorf("Clean(empty) = %q, want empty", got)
	}
}

func TestOverlongRejected(t *testing.T) {
	long := "https://example.com/" + string(make([]byte, 3000))
	for _, kind := range []Kind{SiteLink, External, Image, Contact} {
		if _, ok := Canonicalize(kind, long); ok {
			t.Errorf("kind %d overlong: want invalid", kind)
		}
	}
}
