package forum

import "testing"

func TestIsSafeRedirect(t *testing.T) {
	unsafe := []string{
		"",
		"javascript:alert(1)",
		"https://evil.com",
		"//evil.com",
		`\evil.com`,
		`/\evil.com`,
		`/\/evil.com`,
		`/\\evil.com`,
		`/\.evil.com`,
		" /topics/3",
		"http://evil.com/path",
	}
	for _, input := range unsafe {
		if isSafeRedirect(input) {
			t.Errorf("isSafeRedirect(%q) = true, want false", input)
		}
	}

	safe := []string{
		"/",
		"/topics/3",
		"/p/post/93?page=2",
		"/publish",
		"/u/1?tab=topics",
	}
	for _, input := range safe {
		if !isSafeRedirect(input) {
			t.Errorf("isSafeRedirect(%q) = false, want true", input)
		}
	}
}
