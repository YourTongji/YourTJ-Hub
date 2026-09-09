package markdown2html

import (
	"strings"
	"testing"
)

func TestExtractUsernames(t *testing.T) {
	cases := []struct {
		name     string
		markdown string
		want     []string
	}{
		{
			name:     "plain mention",
			markdown: "hello @alice",
			want:     []string{"alice"},
		},
		{
			name:     "dedupe repeated same username",
			markdown: "hello @alice and @alice again",
			want:     []string{"alice"},
		},
		{
			name:     "multiple unique usernames keep order",
			markdown: "@bob, @alice, @carol",
			want:     []string{"bob", "alice", "carol"},
		},
		{
			name:     "mention at text start",
			markdown: "@alice hello",
			want:     []string{"alice"},
		},
		{
			name:     "inline code is not a mention",
			markdown: "use `@alice` in code",
			want:     nil,
		},
		{
			name:     "fenced code is not a mention",
			markdown: "```\n@alice\n```",
			want:     nil,
		},
		{
			name:     "email at-sign is not a mention",
			markdown: "contact foo@alice.com",
			want:     nil,
		},
		{
			name:     "link destination is not a mention",
			markdown: "[see here](https://example.com/@alice)",
			want:     nil,
		},
		{
			name:     "link text is not a mention",
			markdown: "[@alice](https://example.com)",
			want:     nil,
		},
		{
			name:     "autolink url is not a mention",
			markdown: "see <https://example.com/@alice>",
			want:     nil,
		},
		{
			name:     "image alt is not a mention",
			markdown: "![avatar @alice](/static/pic/1.webp)",
			want:     nil,
		},
		{
			name:     "mention glued to word is not a mention",
			markdown: "foo@alice",
			want:     nil,
		},
		{
			name:     "username with underscore",
			markdown: "hi @alice_smith",
			want:     []string{"alice_smith"},
		},
		{
			name:     "mention followed by punctuation",
			markdown: "@alice, please come",
			want:     []string{"alice"},
		},
		{
			name:     "mention inside bold",
			markdown: "**@alice** is online",
			want:     []string{"alice"},
		},
		{
			name:     "math segment is not a mention",
			markdown: "the value $@alice$ is math",
			want:     nil,
		},
		{
			name:     "empty input",
			markdown: "",
			want:     nil,
		},
		{
			name:     "no at sign",
			markdown: "plain text only",
			want:     nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractUsernames(tc.markdown)
			if len(got) != len(tc.want) {
				t.Fatalf("ExtractUsernames(%q) = %v, want %v", tc.markdown, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("ExtractUsernames(%q) = %v, want %v", tc.markdown, got, tc.want)
				}
			}
		})
	}
}

func TestPostMarkdownToHTMLWithMentions(t *testing.T) {
	targets := map[string]uint64{"alice": 42, "bob": 7}

	cases := []struct {
		name     string
		markdown string
		want     string
		notWant  string
	}{
		{
			name:     "valid mention becomes user link",
			markdown: "hello @alice",
			want:     `<a href="/u/42">@alice</a>`,
		},
		{
			name:     "unknown username stays plain text",
			markdown: "hello @unknown",
			want:     "@unknown",
			notWant:  `<a href="/u/`,
		},
		{
			name:     "inline code keeps plain mention",
			markdown: "use `@alice` in code",
			want:     "<code>@alice</code>",
			notWant:  `<a href="/u/42">`,
		},
		{
			name:     "fenced code keeps plain mention",
			markdown: "```\n@alice\n```",
			want:     "@alice",
			notWant:  `<a href="/u/42">`,
		},
		{
			name:     "email is not linked",
			markdown: "contact foo@alice.com",
			want:     "foo@alice.com",
			notWant:  `<a href="/u/42">`,
		},
		{
			name:     "existing link text is not nested",
			markdown: "[@alice](https://example.com)",
			want:     `href="https://example.com"`,
			notWant:  `<a href="/u/42">`,
		},
		{
			name:     "multiple mentions both link",
			markdown: "@alice and @bob",
			want:     `<a href="/u/42">@alice</a>`,
		},
		{
			name:     "mention in heading keeps anchor",
			markdown: "# @alice heading",
			want:     `<h1 id="alice-heading">`,
		},
		{
			name:     "math segment stays intact",
			markdown: "value $@alice$ is math",
			want:     "@alice",
			notWant:  `<a href="/u/42">`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := PostMarkdownToHTMLWithMentions(tc.markdown, targets)
			if tc.want != "" && !strings.Contains(html, tc.want) {
				t.Fatalf("PostMarkdownToHTMLWithMentions(%q) = %s, want contains %q", tc.markdown, html, tc.want)
			}
			if tc.notWant != "" && strings.Contains(html, tc.notWant) {
				t.Fatalf("PostMarkdownToHTMLWithMentions(%q) = %s, not want contains %q", tc.markdown, html, tc.notWant)
			}
		})
	}
}

func TestPostMarkdownToHTMLWithMentionsNoTargets(t *testing.T) {
	html := PostMarkdownToHTMLWithMentions("hello @alice", map[string]uint64{})
	if strings.Contains(html, `<a href="/u/`) {
		t.Fatalf("empty targets must not produce links, got %s", html)
	}
	if !strings.Contains(html, "@alice") {
		t.Fatalf("mention text must be preserved, got %s", html)
	}
}

func TestPostMarkdownToHTMLWithMentionsXSS(t *testing.T) {
	html := PostMarkdownToHTMLWithMentions(
		"@alice <script>alert(1)</script>",
		map[string]uint64{"alice": 42},
	)
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatalf("script must stay escaped, got %s", html)
	}
	if !strings.Contains(html, `<a href="/u/42">@alice</a>`) {
		t.Fatalf("mention must link, got %s", html)
	}
}