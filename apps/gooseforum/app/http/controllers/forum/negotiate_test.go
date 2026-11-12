package forum

import "testing"

func TestPrefersTextMarkdown(t *testing.T) {
	cases := []struct {
		accept string
		want   bool
	}{
		{accept: "", want: false},
		{accept: "text/html", want: false},
		{accept: "text/markdown", want: true},
		{accept: "text/markdown;charset=utf-8", want: true},
		{accept: "text/markdown, text/html;q=0.8", want: true},
		{accept: "text/html, text/markdown;q=0.1", want: false},
		{accept: "text/html, text/markdown", want: false},
		{accept: "text/markdown;q=0", want: false},
		{accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", want: false},
	}
	for _, testCase := range cases {
		if got := prefersTextMarkdown(testCase.accept); got != testCase.want {
			t.Errorf("prefersTextMarkdown(%q) = %v, want %v", testCase.accept, got, testCase.want)
		}
	}
}
