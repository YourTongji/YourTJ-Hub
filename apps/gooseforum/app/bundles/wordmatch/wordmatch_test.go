package wordmatch

import (
	"fmt"
	"strings"
	"testing"
)

func TestNormalizeNameOptions(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "lowercase", in: "Admin", want: "admin"},
		{name: "full-width ascii", in: "ａｄｍｉｎ", want: "admin"},
		{name: "leet digits", in: "adm1n", want: "admin"},
		{name: "leet 0", in: "r00t", want: "root"},
		{name: "leet 3/5/7", in: "53cur17y", want: "security"},
		{name: "zero width stripped", in: "a\u200bb\u200dc", want: "abc"},
		{name: "zero width bidi", in: "\u200fadmin", want: "admin"},
		{name: "nfkc fullwidth zero width", in: "\u200bａｄｍｉｎ\u200b", want: "admin"},
		{name: "chinese untouched", in: "赌博", want: "赌博"},
		{name: "mapped digits fold only", in: "user_2026", want: "user_2o26"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in, NameOptions)
			if got != tt.want {
				t.Fatalf("Normalize(%q, NameOptions) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeContentOptionsDoesNotLeet(t *testing.T) {
	got := Normalize("STAR 5tar", ContentOptions)
	want := "star 5tar"
	if got != want {
		t.Fatalf("ContentOptions must not fold leet digits: got %q want %q", got, want)
	}
}

func TestNormalizeContentOptionsKeepsCaseAndWidthFolding(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "fullwidth lowercased", in: "ＡＤＭＩＮ", want: "admin"},
		{name: "mixed ascii cn", in: "spammer 内容", want: "spammer 内容"},
		{name: "zero width inside chinese", in: "\u200b赌\u200b博\u200b", want: "赌博"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in, ContentOptions)
			if got != tt.want {
				t.Fatalf("Normalize(%q, ContentOptions) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCompileSkipsEmptyAndDuplicate(t *testing.T) {
	m := Compile([]string{"", "  ", "Admin", "admin", "ＡＤＭＩＮ", "\u200b", "root"}, NameOptions)
	if m.Len() != 2 {
		t.Fatalf("Len() = %d, want 2 (dedup + skip empties)", m.Len())
	}
	if m.words[0].Original != "Admin" {
		t.Fatalf("first original = %q, want %q", m.words[0].Original, "Admin")
	}
	if m.words[1].Original != "root" {
		t.Fatalf("second original = %q, want %q", m.words[1].Original, "root")
	}
}

func TestMatcherEqual(t *testing.T) {
	m := Compile([]string{"Admin", "root", "m1sskey"}, NameOptions)

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "exact lower", in: "admin", want: true},
		{name: "exact mixed", in: "Admin", want: true},
		{name: "leet", in: "adm1n", want: true},
		{name: "leet misskey", in: "m1sskey", want: true},
		{name: "full width", in: "ａｄｍｉｎ", want: true},
		{name: "zero width inside", in: "a\u200bdmin", want: true},
		{name: "no substring", in: "myadmin", want: false},
		{name: "no prefix", in: "administrator", want: false},
		{name: "unrelated", in: "normal_user", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWord, ok := m.Equal(tt.in)
			if ok != tt.want {
				t.Fatalf("Equal(%q) ok = %v, want %v", tt.in, ok, tt.want)
			}
			if tt.want && gotWord == "" {
				t.Fatalf("Equal(%q) returned empty word on hit", tt.in)
			}
		})
	}
}

func TestMatcherEqualEmpty(t *testing.T) {
	m := Compile(nil, NameOptions)
	if _, ok := m.Equal("anything"); ok {
		t.Fatal("Equal on empty matcher must not hit")
	}
}

func TestMatcherFind(t *testing.T) {
	m := Compile([]string{"赌博", "代考", "spammer"}, ContentOptions)

	tests := []struct {
		name     string
		in       string
		wantHit  bool
		wantWord string
	}{
		{name: "first word hit", in: "一起来讨论赌博问题", wantHit: true, wantWord: "赌博"},
		{name: "second word hit", in: "代考服务", wantHit: true, wantWord: "代考"},
		{name: "case insensitive", in: "SPAMMER 内容", wantHit: true, wantWord: "spammer"},
		{name: "full width", in: "全角ＳＰＡＭＭＥＲ", wantHit: true, wantWord: "spammer"},
		{name: "zero width evasion", in: "赌\u200b博", wantHit: true, wantWord: "赌博"},
		{name: "no hit", in: "正常内容", wantHit: false, wantWord: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWord, ok := m.Find(tt.in)
			if ok != tt.wantHit {
				t.Fatalf("Find(%q) ok = %v, want %v", tt.in, ok, tt.wantHit)
			}
			if gotWord != tt.wantWord {
				t.Fatalf("Find(%q) word = %q, want %q", tt.in, gotWord, tt.wantWord)
			}
		})
	}
}

// TestMatcherFindOverlappingAndOrder 失败链会同时报告同一位置结束的重叠词；
// Find 返回编译顺序中第一个命中的词（与旧逐词 Contains 语义一致）。
func TestMatcherFindOverlappingAndOrder(t *testing.T) {
	m := Compile([]string{"abcd", "bc", "xyz"}, ContentOptions)
	word, ok := m.Find("xxabcdyy")
	if !ok {
		t.Fatal("Find(xxabcdyy) no hit")
	}
	if word != "abcd" {
		t.Fatalf("Find(xxabcdyy) = %q, want first-compiled hit abcd", word)
	}
}

func TestMatcherFindAll(t *testing.T) {
	m := Compile([]string{"赌博", "代考", "spammer", "赌博"}, ContentOptions)
	hits := m.FindAll("SPAMMER 代考 赌博 内容")
	want := []string{"赌博", "代考", "spammer"} // 编译顺序 + 去重
	if len(hits) != len(want) {
		t.Fatalf("FindAll hits = %v, want %v", hits, want)
	}
	for i := range want {
		if hits[i] != want[i] {
			t.Fatalf("FindAll hits = %v, want %v", hits, want)
		}
	}
}

func TestMatcherFindAllOverlapping(t *testing.T) {
	m := Compile([]string{"abcd", "bc"}, ContentOptions)
	hits := m.FindAll("xabcd")
	if len(hits) != 2 {
		t.Fatalf("overlapping FindAll = %v, want both abcd and bc", hits)
	}
}

func TestMatcherFindEmptyAndNil(t *testing.T) {
	var nilM *Matcher
	if _, ok := nilM.Find("x"); ok {
		t.Fatal("nil matcher Find must not hit")
	}
	empty := Compile(nil, ContentOptions)
	if _, ok := empty.Find("x"); ok {
		t.Fatal("empty matcher Find must not hit")
	}
}

func TestMatcherFindAllNilForNoHit(t *testing.T) {
	m := Compile([]string{"赌博"}, ContentOptions)
	if hits := m.FindAll("正常内容"); hits != nil {
		t.Fatalf("FindAll no-hit = %v, want nil", hits)
	}
}

// 生成确定性基准词库：约 1000 词（中英混合），避免随机导致的抖动。
func benchWords(n int) []string {
	words := make([]string, 0, n)
	candidates := []string{"代考", "赌博", "色情", "诈骗", "毒品"}
	for i := 0; i < n; i++ {
		switch i % 5 {
		case 0:
			words = append(words, fmt.Sprintf("word%d", i))
		case 1:
			words = append(words, fmt.Sprintf("屏蔽词%d号", i))
		case 2:
			words = append(words, fmt.Sprintf("term%d", i))
		case 3:
			words = append(words, candidates[len(words)%len(candidates)])
		default:
			words = append(words, fmt.Sprintf("x%dblocked", i))
		}
	}
	return words
}

func benchHaystack(size int) string {
	base := "这是一段普通的论坛讨论内容，包含正常词汇和 English words for scanning overhead。"
	var b strings.Builder
	for b.Len() < size {
		b.WriteString(base)
	}
	return b.String()
}

func BenchmarkCompile1000Words(b *testing.B) {
	words := benchWords(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := Compile(words, ContentOptions)
		if m.Len() == 0 {
			b.Fatal("compile produced empty matcher")
		}
	}
}

func BenchmarkFind1000Words50kChars(b *testing.B) {
	m := Compile(benchWords(1000), ContentOptions)
	haystack := benchHaystack(50_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := m.Find(haystack); ok {
			b.Fatal("unexpected hit")
		}
	}
}

func BenchmarkFindHit1000Words50kChars(b *testing.B) {
	m := Compile(benchWords(1000), ContentOptions)
	haystack := benchHaystack(50_000) + " 这里有一个诈骗词"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := m.Find(haystack); !ok {
			b.Fatal("expected hit")
		}
	}
}
