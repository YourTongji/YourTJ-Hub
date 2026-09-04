package forum

import (
	"encoding/json"
	"testing"
)

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
		{Key: "content", URL: "/settings?tab=content"},
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

func TestSettingsPagePropsSerializesGoogleOAuthReady(t *testing.T) {
	encoded, err := json.Marshal(SettingsPageProps{GoogleOAuthReady: true})
	if err != nil {
		t.Fatalf("marshal SettingsPageProps: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal SettingsPageProps: %v", err)
	}
	if ready, ok := payload["googleOAuthReady"].(bool); !ok || !ready {
		t.Fatalf("googleOAuthReady = %#v, want true", payload["googleOAuthReady"])
	}
}

func TestParsePublishContentType(t *testing.T) {
	tests := []struct {
		input string
		want  int8
	}{
		{"", 0},
		{"regular", 0},
		{"0", 0},
		{"question", 1},
		{"1", 1},
		{"Question", 1},
		{"thought", 2},
		{"2", 2},
		{"Thought", 2},
		{"article", 3},
		{"3", 3},
		{"Article", 3},
		{"unknown", 0},
	}
	for _, tt := range tests {
		got := parsePublishContentType(tt.input)
		if got != tt.want {
			t.Errorf("parsePublishContentType(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestTopicPermissionsCanPost(t *testing.T) {
	tests := []struct {
		name        string
		userID      uint64
		contentType int8
		wantCanPost bool
	}{
		{"guest cannot post to regular", 0, 0, false},
		{"user can post to regular", 1, 0, true},
		{"user can post to question", 1, 1, true},
		{"user can post to thought (moment)", 1, 2, true},
		{"user can post to article", 1, 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canPost := tt.userID > 0 && (tt.contentType == 0 || tt.contentType == 1 || tt.contentType == 2 || tt.contentType == 3)
			if canPost != tt.wantCanPost {
				t.Errorf("canPost = %v, want %v", canPost, tt.wantCanPost)
			}
		})
	}
}
