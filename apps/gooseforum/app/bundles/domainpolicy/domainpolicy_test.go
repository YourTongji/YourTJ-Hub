package domainpolicy

import (
	"fmt"
	"strings"
	"testing"
)

func TestCanonicalizeEntry(t *testing.T) {
	valid := map[string]string{
		"github.com":        "github.com",
		"  GitHub.COM  ":    "github.com",
		"bilibili.com.":     "bilibili.com",
		"mp.weixin.qq.com":  "mp.weixin.qq.com",
		"中国.com":            "xn--fiqs8s.com",
		"xn--fiqs8s.com":    "xn--fiqs8s.com",
		"a-b.example.co.uk": "a-b.example.co.uk",
		"zh.wikipedia.org":  "zh.wikipedia.org",
	}
	for raw, want := range valid {
		got, ok := Canonicalize(raw)
		if !ok {
			t.Errorf("Canonicalize(%q) = invalid, want valid", raw)
			continue
		}
		if got != want {
			t.Errorf("Canonicalize(%q) = %q, want %q", raw, got, want)
		}
	}

	// Blank entries are valid and collapse to "" (paste noise is skipped, not
	// rejected).
	if got, ok := Canonicalize("   "); !ok || got != "" {
		t.Errorf("Canonicalize(blank) = (%q, %v), want (\"\", true)", got, ok)
	}

	invalid := []string{
		"https://github.com",
		"http://github.com",
		"github.com/path",
		"github.com:443",
		"github.com?q=1",
		"github.com#frag",
		"*.github.com",
		"github .com",
		"user@github.com",
		"github,com",
		"github\u0000.com",
		"java\nscript:alert(1)",
		"localhost",
		"com",
		"co.uk",
		"github.io",
		"192.168.1.1",
		"[::1]",
		"-github.com",
		"github-.com",
		"github..com",
		".github.com",
		strings.Repeat("a", maxLabelLength+1) + ".com",
		strings.Repeat("a", maxDomainLength) + ".com",
	}
	for _, raw := range invalid {
		if got, ok := Canonicalize(raw); ok {
			t.Errorf("Canonicalize(%q) = (%q, valid), want invalid", raw, got)
		}
	}
}

func TestCanonicalizeListKeepsOrderAndReportsInvalid(t *testing.T) {
	kept, dropped := CanonicalizeList([]string{
		"GitHub.com",
		"",
		"  ",
		"github.com",
		"https://bilibili.com",
		"bilibili.com",
		"mp.weixin.qq.com",
	})
	want := []string{"github.com", "bilibili.com", "mp.weixin.qq.com"}
	if len(kept) != len(want) {
		t.Fatalf("kept = %q, want %q", kept, want)
	}
	for i := range want {
		if kept[i] != want[i] {
			t.Fatalf("kept = %q, want %q", kept, want)
		}
	}
	if len(dropped) != 1 || dropped[0] != "https://bilibili.com" {
		t.Fatalf("dropped = %q, want the scheme-carrying entry", dropped)
	}
}

func TestCanonicalizeListCapsEntries(t *testing.T) {
	entries := make([]string, 0, MaxEntries+1)
	for i := 0; i <= MaxEntries; i++ {
		entries = append(entries, fmt.Sprintf("site%d.example.com", i))
	}
	kept, dropped := CanonicalizeList(entries)
	if len(kept) != MaxEntries {
		t.Fatalf("kept %d entries, want %d", len(kept), MaxEntries)
	}
	if len(dropped) != 1 || dropped[0] != entries[MaxEntries] {
		t.Fatalf("dropped = %q, want the entry beyond the cap", dropped)
	}
}

func TestCanonicalizeListEmptyInput(t *testing.T) {
	kept, dropped := CanonicalizeList(nil)
	if len(kept) != 0 || len(dropped) != 0 {
		t.Fatalf("CanonicalizeList(nil) = (%q, %q), want empty", kept, dropped)
	}
}
