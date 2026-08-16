package forum

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
