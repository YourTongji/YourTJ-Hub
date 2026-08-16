package forum

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gin-gonic/gin"
)

func TestWikiDetailServesRepositoryAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := t.TempDir()
	asset := filepath.Join(repo, "assets", "guide.pdf")
	if err := os.MkdirAll(filepath.Dir(asset), 0o755); err != nil {
		t.Fatalf("create asset directory: %v", err)
	}
	if err := os.WriteFile(asset, []byte("PDF"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	previousRepo := preferences.GetString("wiki.git.repo", "")
	previousCloneDir := preferences.GetString("wiki.git.clone_dir", "")
	preferences.Set("wiki.git.repo", "https://github.com/YourTongji/YourTJ-Wiki.git")
	preferences.Set("wiki.git.clone_dir", repo)
	t.Cleanup(func() {
		preferences.Set("wiki.git.repo", previousRepo)
		preferences.Set("wiki.git.clone_dir", previousCloneDir)
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/wiki/_assets/assets/guide.pdf", nil)
	context.Params = gin.Params{{Key: "path", Value: "/_assets/assets/guide.pdf"}}
	WikiDetail(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "PDF" {
		t.Fatalf("asset body = %q, want PDF", recorder.Body.String())
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", recorder.Header().Get("X-Content-Type-Options"))
	}
}

// TestWikiAssetTypeAllowlist review H1：白名单类型内联渲染，危险/未知类型
// 一律拒绝内联（返回 ok=false → octet-stream + attachment）。
func TestWikiAssetTypeAllowlist(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		wantOK bool
		wantCT string
	}{
		{"png", "a.png", true, "image/png"},
		{"jpeg", "a.jpg", true, "image/jpeg"},
		{"webp", "a.webp", true, "image/webp"},
		{"pdf", "a.pdf", true, "application/pdf"},
		{"docx", "a.docx", true, "application/msword"},
		{"zip", "a.zip", true, "application/octet-stream"},
		{"txt", "a.txt", true, "text/plain"},
		{"html rejected", "a.html", false, ""},
		{"svg rejected", "a.svg", false, ""},
		{"js rejected", "a.js", false, ""},
		{"wasm rejected", "a.wasm", false, ""},
		{"md rejected", "a.md", false, ""},
		{"markdown rejected", "a.markdown", false, ""},
		{"extensionless rejected", "README", false, ""},
		{"unknown ext rejected", "a.xyz", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ct, ok := wikiAssetType(tc.file)
			if ok != tc.wantOK {
				t.Fatalf("wikiAssetType(%q) ok=%v, want %v", tc.file, ok, tc.wantOK)
			}
			if ok && ct != tc.wantCT {
				t.Fatalf("wikiAssetType(%q) ct=%q, want %q", tc.file, ct, tc.wantCT)
			}
		})
	}
}

// TestWikiDetailServesHtmlAsAttachment review H1：即使仓库里有 .html 文件，
// 服务端也必须以 octet-stream + attachment 返回，绝不内联渲染（防同源 XSS）。
func TestWikiDetailServesHtmlAsAttachment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := t.TempDir()
	asset := filepath.Join(repo, "assets", "evil.html")
	if err := os.MkdirAll(filepath.Dir(asset), 0o755); err != nil {
		t.Fatalf("create asset directory: %v", err)
	}
	if err := os.WriteFile(asset, []byte("<script>alert(1)</script>"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	previousRepo := preferences.GetString("wiki.git.repo", "")
	previousCloneDir := preferences.GetString("wiki.git.clone_dir", "")
	preferences.Set("wiki.git.repo", "https://github.com/YourTongji/YourTJ-Wiki.git")
	preferences.Set("wiki.git.clone_dir", repo)
	t.Cleanup(func() {
		preferences.Set("wiki.git.repo", previousRepo)
		preferences.Set("wiki.git.clone_dir", previousCloneDir)
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/wiki/_assets/assets/evil.html", nil)
	context.Params = gin.Params{{Key: "path", Value: "/_assets/assets/evil.html"}}
	WikiDetail(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if ct := recorder.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("Content-Type = %q, want application/octet-stream", ct)
	}
	if cd := recorder.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Fatalf("Content-Disposition = %q, want attachment", cd)
	}
	if csp := recorder.Header().Get("Content-Security-Policy"); csp != "sandbox" {
		t.Fatalf("Content-Security-Policy = %q, want sandbox", csp)
	}
}
