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
		{Key: "deleted", URL: "/settings?tab=deleted"},
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

	// Invariant assertions: lock the contract beyond the exact-match list above.
	activeCount := 0
	seen := make(map[string]bool, len(got))
	for _, tab := range got {
		if tab.Active {
			activeCount++
		}
		if seen[tab.Key] {
			t.Errorf("settingsTabs() has duplicate key %q", tab.Key)
		}
		seen[tab.Key] = true

		if tab.Key == "profile" {
			if tab.URL != "/settings" {
				t.Errorf("settingsTabs() profile URL = %q, want /settings", tab.URL)
			}
		} else if wantURL := "/settings?tab=" + tab.Key; tab.URL != wantURL {
			t.Errorf("settingsTabs() %q URL = %q, want %q", tab.Key, tab.URL, wantURL)
		}
	}
	if activeCount != 1 {
		t.Errorf("settingsTabs() active tab count = %d, want exactly 1", activeCount)
	}
}
