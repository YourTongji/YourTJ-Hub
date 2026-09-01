package wikiservice

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenWikiAsset(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "assets/guide.pdf", "PDF")

	file, info, err := OpenWikiAsset(repo, "assets/guide.pdf")
	if err != nil {
		t.Fatalf("open asset: %v", err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if info.Name() != "guide.pdf" {
		t.Fatalf("asset name = %q, want guide.pdf", info.Name())
	}
	body, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read asset: %v", err)
	}
	if string(body) != "PDF" {
		t.Fatalf("asset body = %q, want PDF", body)
	}

	for _, path := range []string{"../outside.txt", "assets/guide.md", "assets/missing.pdf"} {
		if _, _, err := OpenWikiAsset(repo, path); err == nil {
			t.Fatalf("OpenWikiAsset(%q) succeeded, want rejection", path)
		}
	}

	outside := filepath.Join(t.TempDir(), "outside.pdf")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside asset: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "assets", "outside.pdf")); err != nil {
		t.Fatalf("create asset symlink: %v", err)
	}
	if _, _, err := OpenWikiAsset(repo, "assets/outside.pdf"); err == nil || !strings.Contains(err.Error(), "outside repository") {
		t.Fatalf("symlink escape error = %v, want outside-repository rejection", err)
	}
}

// TestSplitRepoOwnerName review P2：仅接受 github.com 主机且路径恰好为
// owner/repo 两段；ssh://、非 GitHub 主机、多余路径段一律返回空
// （jsDelivr 模式退回 self 路由，绝不产出畸形 CDN 链接）。
func TestSplitRepoOwnerName(t *testing.T) {
	cases := []struct {
		repo      string
		wantOwner string
		wantName  string
	}{
		{"https://github.com/YourTongji/YourTJ-Wiki.git", "YourTongji", "YourTJ-Wiki"},
		{"https://github.com/YourTongji/YourTJ-Wiki", "YourTongji", "YourTJ-Wiki"},
		{"http://github.com/YourTongji/YourTJ-Wiki.git", "YourTongji", "YourTJ-Wiki"},
		{"git@github.com:YourTongji/YourTJ-Wiki.git", "YourTongji", "YourTJ-Wiki"},
		{"git@github.com:YourTongji/YourTJ-Wiki", "YourTongji", "YourTJ-Wiki"},
		// 拒绝：非 GitHub 主机 / 非 https/http/ssh 形式 / 多余路径段 / 空。
		{"ssh://git@github.com/YourTongji/YourTJ-Wiki.git", "", ""},
		{"git://github.com/YourTongji/YourTJ-Wiki.git", "", ""},
		{"file:///srv/wiki.git", "", ""},
		{"https://gitlab.com/YourTongji/YourTJ-Wiki.git", "", ""},
		{"https://github.com/YourTongji/YourTJ-Wiki/extra", "", ""},
		{"https://github.com/YourTongji/YourTJ-Wiki.git?tab=readme", "", ""},
		{"github.com/YourTongji/YourTJ-Wiki", "", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		owner, name := splitRepoOwnerName(tc.repo)
		if owner != tc.wantOwner || name != tc.wantName {
			t.Errorf("splitRepoOwnerName(%q) = (%q, %q), want (%q, %q)",
				tc.repo, owner, name, tc.wantOwner, tc.wantName)
		}
	}
}
