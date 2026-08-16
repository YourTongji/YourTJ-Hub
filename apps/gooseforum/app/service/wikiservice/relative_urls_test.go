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
