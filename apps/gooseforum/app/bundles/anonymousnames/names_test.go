package anonymousnames

import (
	"testing"
	"unicode"
)

func assertPhrase(t *testing.T, name string) {
	t.Helper()
	runes := []rune(name)
	if len(runes) != 6 || runes[4] != '的' {
		t.Fatalf("expected six-character phrase, got %q", name)
	}
	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			t.Fatalf("non-Han name: %q", name)
		}
	}
}

func TestBatchSixCharacterNames(t *testing.T) {
	for range 20 {
		batch, err := Batch()
		if err != nil {
			t.Fatal(err)
		}
		if len(batch) != 10 {
			t.Fatalf("batch size: %d", len(batch))
		}
		seen := make(map[string]bool)
		for _, name := range batch {
			assertPhrase(t, name)
			if seen[name] {
				t.Fatalf("repeated candidate: %q", name)
			}
			seen[name] = true
		}
	}
}

func TestCuratedCombinations(t *testing.T) {
	pool, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if pool.size() < 100000 {
		t.Fatalf("insufficient variety: %d", pool.size())
	}
	seen := make(map[string]bool)
	for index := range pool.size() {
		name := pool.name(index)
		assertPhrase(t, name)
		if seen[name] {
			t.Fatalf("duplicate combination: %q", name)
		}
		seen[name] = true
	}
	for _, example := range []string{"躲进云里的猫", "抱着松果的熊", "躲进松果的猫"} {
		if !seen[example] {
			t.Fatalf("missing free combination: %q", example)
		}
	}
}
