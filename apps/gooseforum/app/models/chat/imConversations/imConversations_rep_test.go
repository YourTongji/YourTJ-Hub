package imConversations

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestMessagePreviewStripsReplyQuote(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "reply preview shows the body",
			content: "> @bob: 你好\n\n收到",
			want:    "收到",
		},
		{
			name:    "quote without a body keeps its text",
			content: "> @bob: 你好",
			want:    "> @bob: 你好",
		},
		{
			name:    "plain content is unchanged",
			content: "普通消息",
			want:    "普通消息",
		},
		{
			name:    "body keeps its own quote-like lines",
			content: "> @bob: 你好\n\n甲\n\n> 乙",
			want:    "甲\n\n> 乙",
		},
		{
			name:    "quote marker without a blank line is unchanged",
			content: "> 普通的一行\n第二行",
			want:    "> 普通的一行\n第二行",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MessagePreview(test.content); got != test.want {
				t.Fatalf(
					"MessagePreview(%q) = %q, want %q",
					test.content,
					got,
					test.want,
				)
			}
		})
	}
}

func TestMessagePreviewBoundsQuotedReplyBody(t *testing.T) {
	content := "> @bob: " + strings.Repeat("引", 200) + "\n\n" +
		strings.Repeat("回", 300)
	got := MessagePreview(content)
	if utf8.RuneCountInString(got) > 255 {
		t.Fatalf("preview exceeds 255 runes: %d", utf8.RuneCountInString(got))
	}
	if strings.Contains(got, "> @bob") {
		t.Fatalf("preview kept the quote header: %q", got)
	}
	if !strings.HasPrefix(got, strings.Repeat("回", 10)) {
		t.Fatalf("preview lost the reply body: %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated preview has no truncation marker: %q", got)
	}
}
