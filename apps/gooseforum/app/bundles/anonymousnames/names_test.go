package anonymousnames

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCompleteUnfilteredLexicon(t *testing.T) {
	words, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 156289 {
		t.Fatalf("distinct words: %d", len(words))
	}
	min, max, symbols := 100, 0, false
	for _, word := range words {
		n := utf8.RuneCountInString(word)
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
		if strings.Contains(word, "++") {
			symbols = true
		}
	}
	if min != 1 || max != 29 || !symbols {
		t.Fatalf("filtered lexicon: min=%d max=%d symbols=%v", min, max, symbols)
	}
	eligible := make(map[string]bool, len(words))
	for _, word := range words {
		eligible[word] = true
	}
	for i := 0; i < 20; i++ {
		batch, err := Batch()
		if err != nil {
			t.Fatal(err)
		}
		seen := make(map[string]bool)
		if len(batch) != 10 {
			t.Fatal(batch)
		}
		for _, word := range batch {
			if !eligible[word] || seen[word] {
				t.Fatalf("invalid candidate: %q", word)
			}
			seen[word] = true
		}
	}
}
