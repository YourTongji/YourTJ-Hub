package i18n

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"":               Fallback,
		"zh":             "zh",
		"ZH":             "zh",
		"zh-CN":          "zh",
		"en":             "en",
		"en-US,en;q=0.9": "en",
		"ja_JP":          "ja",
		"de":             "de",
		"de-DE":          "de",
		"ko":             Fallback, // unsupported -> fallback
		"  en  ":         "en",
		"fr-FR,fr;q=0.7": Fallback,
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTranslateFallback(t *testing.T) {
	// A key present in every locale returns the localized value.
	if got := T("en", "search"); got != "Search" {
		t.Errorf(`T("en","search") = %q, want "Search"`, got)
	}
	if got := T("de", "search"); got != "Suchen" {
		t.Errorf(`T("de","search") = %q, want "Suchen"`, got)
	}
	// An unsupported locale falls back to zh (the source locale).
	if got := T("ko", "search"); got != "搜索" {
		t.Errorf(`T("ko","search") = %q, want the zh fallback`, got)
	}
	// An unknown key returns the key itself (visible, never blank).
	if got := T("en", "does.not.exist"); got != "does.not.exist" {
		t.Errorf(`T for missing key = %q, want the key`, got)
	}
}

func TestTranslateInterpolation(t *testing.T) {
	got := T("en", "meta.searchDesc", "site", "GooseForum")
	want := "Search topics, keywords, and discussions on GooseForum."
	if got != want {
		t.Errorf("interpolated = %q, want %q", got, want)
	}
	// Numeric params render without quotes.
	if got := T("en", "partnersSummary", "count", 7); got != "7 community partners." {
		t.Errorf("partnersSummary = %q", got)
	}
}

func TestFunc(t *testing.T) {
	tr := Func("ja")
	if got := tr("search"); got != "検索" {
		t.Errorf(`Func("ja")("search") = %q, want "検索"`, got)
	}
}

func TestServerMessage(t *testing.T) {
	fallback := "Failed to load"
	cases := []struct {
		name     string
		lang     string
		code     string
		params   map[string]any
		want     string
		contains string
		notWant  string
	}{
		{
			name: "flat serverMessages key",
			lang: "en",
			code: "topic.notFound",
			want: "The topic does not exist or has been deleted.",
		},
		{
			// 该码仅存在于 dotted `server.<code>` 命名空间(无 flat
			// serverMessages.* 覆盖),专门验证备用分支真实命中。
			name: "dotted server key only",
			lang: "en",
			code: "common.request.invalidFormat",
			want: "The request format is invalid. Please check and try again.",
		},
		{
			name: "flat serverMessages key wins over dotted",
			lang: "en",
			code: "common.operation.failed",
			want: "Operation failed. Please try again later.",
		},
		{
			name: "unknown code falls back",
			lang: "en",
			code: "some.unknown.code",
			want: fallback,
		},
		{
			name: "empty code falls back",
			lang: "en",
			code: "",
			want: fallback,
		},
		{
			name:     "unknown code never leaks raw code",
			lang:     "en",
			code:     "some.unknown.code",
			want:     fallback,
			notWant:  "some.unknown.code",
		},
		{
			name:     "unsupported lang falls back to zh",
			lang:     "ko",
			code:     "topic.notFound",
			contains: "话题不存在",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := ServerMessage(tt.lang, tt.code, tt.params, fallback)
			if tt.want != "" && got != tt.want {
				t.Errorf("ServerMessage(%q, %q) = %q, want %q", tt.lang, tt.code, got, tt.want)
			}
			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("ServerMessage(%q, %q) = %q, want it to contain %q", tt.lang, tt.code, got, tt.contains)
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("ServerMessage(%q, %q) = %q, must not contain %q", tt.lang, tt.code, got, tt.notWant)
			}
		})
	}
}

func TestServerMessageInterpolation(t *testing.T) {
	// common.rateLimited carries a {retryAfterSeconds} placeholder.
	got := ServerMessage("en", "common.rateLimited", map[string]any{"retryAfterSeconds": 30}, "fallback")
	want := "Too many attempts. Please try again in 30 seconds."
	if got != want {
		t.Errorf("ServerMessage interpolated = %q, want %q", got, want)
	}
}
