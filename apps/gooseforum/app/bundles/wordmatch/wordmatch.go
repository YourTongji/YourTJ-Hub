// Package wordmatch provides a small, dependency-light multi-pattern
// matcher for forum moderation lists (reserved/banned usernames and
// sensitive-word content scanning).
//
// Design goals:
//   - Pure, testable functions. Compiling a word list and matching a haystack
//     have no side effects and no global state, so callers decide when and how
//     often to compile (the moderation policy layer recompiles per check;
//     lists are small and compile is sub-millisecond).
//   - Deterministic normalization pipeline instead of unbounded "fuzzy"
//     matching: case folding, NFKC (full-width/half-width), zero-width
//     stripping, and optional leetspeak folding for ASCII words. Chinese text
//     is left intact by NFKC, so content lists stay precise.
//   - Aho-Corasick automaton for O(haystack + matches) scans, so content
//     checks stay fast even with thousands of words (acceptance: 1,000 words
//     x 50,000 chars well under 3 ms).
//   - The matcher reports the original (pre-normalization) word that hit, so
//     callers can show/log the configured term rather than its folded form.
//
// Leetspeak folding (NameOptions) is only meaningful for whole-string list
// equality (usernames/nicknames); it intentionally never runs on content
// substring scans, where it would over-match ordinary prose.
package wordmatch

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Options selects which normalization passes run before matching.
type Options struct {
	CaseFold       bool // Unicode simple case folding (strings.ToLower)
	NFKC           bool // NFKC: full-width → half-width + compatibility forms
	StripZeroWidth bool // remove U+200B..U+200F, U+FEFF, U+2060..U+2064
	LeetFold       bool // ASCII digit→letter folding: 1→i 0→o 3→e 4→a 5→s 7→t
}

// NameOptions is the normalization used for whole-string list equality
// (reserved/banned usernames and nicknames).
var NameOptions = Options{
	CaseFold:       true,
	NFKC:           true,
	StripZeroWidth: true,
	LeetFold:       true,
}

// ContentOptions is the normalization used for content substring scans.
// Leetspeak folding is intentionally absent: "a 5tar" folding to "star"
// would over-match ordinary prose.
var ContentOptions = Options{
	CaseFold:       true,
	NFKC:           true,
	StripZeroWidth: true,
	LeetFold:       false,
}

// Normalize applies the selected passes to s. It is exported so callers can
// pre-normalize whole strings for equality checks (list membership).
func Normalize(s string, o Options) string {
	if o.NFKC {
		s = norm.NFKC.String(s)
	}
	if o.StripZeroWidth {
		s = stripZeroWidth(s)
	}
	if o.CaseFold {
		s = strings.ToLower(s)
	}
	if o.LeetFold {
		s = leetFold(s)
	}
	return s
}

// stripZeroWidth removes zero-width formatting characters that are commonly
// inserted to defeat substring filters.
func stripZeroWidth(s string) string {
	if !strings.ContainsFunc(s, isZeroWidth) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isZeroWidth(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isZeroWidth reports whether r is a zero-width format character.
func isZeroWidth(r rune) bool {
	switch r {
	case '\u200b', '\u200c', '\u200d', '\u200e', '\u200f', '\ufeff',
		'\u2060', '\u2061', '\u2062', '\u2063', '\u2064':
		return true
	}
	return false
}

// leetFold maps leetspeak digits to the letters they commonly replace. It is
// byte-wise over the already-lowercased ASCII string; bytes outside the
// mapped set are preserved.
func leetFold(s string) string {
	if !strings.ContainsAny(s, "103457") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '1':
			b.WriteByte('i')
		case '0':
			b.WriteByte('o')
		case '3':
			b.WriteByte('e')
		case '4':
			b.WriteByte('a')
		case '5':
			b.WriteByte('s')
		case '7':
			b.WriteByte('t')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// Word is a compiled dictionary entry: the original word and its normalized
// form (empty when the original normalized to nothing, e.g. all zero-width).
type Word struct {
	Original   string
	Normalized string
}

// Compile builds a matcher from a word list. Empty words and words that
// normalize to nothing are skipped; duplicates (after normalization) are
// collapsed, keeping the first original spelling. The returned matcher is
// immutable and safe for concurrent use.
func Compile(words []string, o Options) *Matcher {
	m := &Matcher{
		options:     o,
		words:       make([]Word, 0, len(words)),
		equalLookup: make(map[string]string, len(words)),
	}
	seen := make(map[string]struct{}, len(words))
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		n := Normalize(w, o)
		if n == "" {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		entry := Word{Original: w, Normalized: n}
		m.words = append(m.words, entry)
		m.equalLookup[n] = w
		m.insert(entry)
	}
	m.buildFails()
	return m
}

// Matcher is an immutable multi-pattern matcher over normalized text.
// Find/FindAll match substrings; Equal matches whole strings (used for
// usernames, which must never match as a substring of a longer name).
type Matcher struct {
	options Options
	words   []Word

	// equalLookup maps a normalized word to its original spelling for
	// whole-string equality checks.
	equalLookup map[string]string

	// root of the Aho-Corasick automaton. nil when the word list is empty.
	root *node
}

// node is one trie node of the Aho-Corasick automaton. A word match is
// reported by walking the failure chain at each position; node.original is
// set for trie-terminal words to short-circuit that walk.
type node struct {
	children map[rune]*node
	fail     *node
	original string // configured word ending exactly at this node, or ""
}

// newTrieNode allocates a node (children allocated lazily).
func newTrieNode() *node {
	return &node{}
}

// insert adds one normalized word to the trie.
func (m *Matcher) insert(w Word) {
	if m.root == nil {
		m.root = newTrieNode()
	}
	cur := m.root
	for _, r := range w.Normalized {
		next := cur.children[r]
		if next == nil {
			next = newTrieNode()
			if cur.children == nil {
				cur.children = make(map[rune]*node, 1)
			}
			cur.children[r] = next
		}
		cur = next
	}
	if cur.original == "" {
		cur.original = w.Original
	}
}

// buildFails computes failure links breadth-first. Words that share a suffix
// surface through the failure chain at scan time, so no output-link
// propagation is needed here.
func (m *Matcher) buildFails() {
	if m.root == nil {
		return
	}
	queue := make([]*node, 0, len(m.words)*4)
	for _, child := range m.root.children {
		child.fail = m.root
		queue = append(queue, child)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for r, child := range cur.children {
			queue = append(queue, child)
			fail := cur.fail
			for fail != nil {
				if next := fail.children[r]; next != nil {
					child.fail = next
					break
				}
				fail = fail.fail
			}
			if fail == nil {
				child.fail = m.root
			}
		}
	}
}

// Words returns the compiled entries (original + normalized) for tests and
// introspection. The returned slice must not be mutated.
func (m *Matcher) Words() []Word {
	if m == nil {
		return nil
	}
	return m.words
}

// Len returns the number of compiled (deduplicated) words.
func (m *Matcher) Len() int {
	if m == nil {
		return 0
	}
	return len(m.words)
}

// Equal reports whether the whole normalized haystack equals any word. It is
// the correct check for list equality (usernames/nicknames), where a
// substring match would wrongly reject "myadmin".
func (m *Matcher) Equal(s string) (string, bool) {
	if m == nil || len(m.equalLookup) == 0 {
		return "", false
	}
	original, ok := m.equalLookup[Normalize(s, m.options)]
	return original, ok
}

// scan walks the normalized haystack once, recording every matched word's
// original spelling into found (deduplicated). Overlapping words that end at
// the same position are all reported via the failure chain.
func (m *Matcher) scan(n string, found map[string]struct{}) {
	if m.root == nil || n == "" {
		return
	}
	cur := m.root
	for _, r := range n {
		for cur != m.root && cur.children[r] == nil {
			cur = cur.fail
		}
		if next := cur.children[r]; next != nil {
			cur = next
		}
		for t := cur; t != m.root; t = t.fail {
			if t.original != "" {
				found[t.original] = struct{}{}
			}
		}
	}
}

// Find scans the haystack once and returns the first configured word whose
// normalized form appears in the normalized haystack, in the order words
// were compiled (matching the previous line-by-line policy semantics). ok is
// false when nothing matched.
func (m *Matcher) Find(s string) (string, bool) {
	if m == nil || len(m.words) == 0 {
		return "", false
	}
	found := make(map[string]struct{}, 4)
	m.scan(Normalize(s, m.options), found)
	for _, w := range m.words {
		if _, ok := found[w.Original]; ok {
			return w.Original, true
		}
	}
	return "", false
}

// FindAll returns every distinct configured word that matches the haystack,
// in compile order. Returns nil when nothing matched.
func (m *Matcher) FindAll(s string) []string {
	if m == nil || len(m.words) == 0 {
		return nil
	}
	found := make(map[string]struct{}, len(m.words))
	m.scan(Normalize(s, m.options), found)
	if len(found) == 0 {
		return nil
	}
	hits := make([]string, 0, len(found))
	for _, w := range m.words {
		if _, ok := found[w.Original]; ok {
			hits = append(hits, w.Original)
		}
	}
	return hits
}
