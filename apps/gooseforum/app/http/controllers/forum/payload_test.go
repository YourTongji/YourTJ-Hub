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

func TestSettingsTabs(t *testing.T) {
	got := settingsTabs()
	want := []TabPayload{
		{Key: "profile", URL: "/settings", Active: true},
		{Key: "account", URL: "/settings?tab=account"},
		{Key: "privacy", URL: "/settings?tab=privacy"},
		{Key: "binding", URL: "/settings?tab=binding"},
		{Key: "security", URL: "/settings?tab=security"},
		{Key: "general", URL: "/settings?tab=general"},
	}
	if len(got) != len(want) {
		t.Fatalf("settingsTabs() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("settingsTabs()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
