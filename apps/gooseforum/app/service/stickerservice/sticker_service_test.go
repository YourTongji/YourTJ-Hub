package stickerservice

import (
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
)

func TestValidateName(t *testing.T) {
	valid := []string{"a", "滑稽", "长_名-字2", strings.Repeat("长", 64)}
	invalid := []string{"", "带 空格", "带:冒号", "带[方括号]", "带]括号", strings.Repeat("长", 65), "带/斜杠"}
	for _, name := range valid {
		if !ValidateName(name) {
			t.Fatalf("ValidateName(%q) = false, want true", name)
		}
	}
	for _, name := range invalid {
		if ValidateName(name) {
			t.Fatalf("ValidateName(%q) = true, want false", name)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"普通名字":    "普通名字",
		"带 空格 名":  "带-空格-名",
		"标点，混排!":  "标点-混排",
		"  trim ": "trim",
		"a:b]c":   "a-b-c",
	}
	for raw, want := range cases {
		if got := SanitizeName(raw); got != want {
			t.Fatalf("SanitizeName(%q) = %q, want %q", raw, got, want)
		}
	}
	if got := SanitizeName(":"); got != "" {
		t.Fatalf("SanitizeName(bare punctuation) = %q, want empty", got)
	}
	long := strings.Repeat("长", 80)
	got := SanitizeName(long)
	if runeCount := len([]rune(got)); runeCount != sticker.MaxNameLen {
		t.Fatalf("SanitizeName long rune count = %d, want %d", runeCount, sticker.MaxNameLen)
	}
}

func TestStemName(t *testing.T) {
	cases := map[string]string{
		"滑稽.png":      "滑稽",
		"dir/内/哈.jpg": "哈",
		"noext":       "noext",
		"多.个.点.gif":   "多.个.点",
		".hidden":     ".hidden",
	}
	for input, want := range cases {
		if got := StemName(input); got != want {
			t.Fatalf("StemName(%q) = %q, want %q", input, got, want)
		}
	}
}
