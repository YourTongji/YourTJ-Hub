package controllers

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestActivationContinuationOnlyNavigatesToForumPages(t *testing.T) {
	for _, target := range []string{"/publish", "/publish?id=42", "/p/post/12/3#post-7", "/campus", "/settings?tab=profile", "/courses/12"} {
		if got := activationContinuation(target); got != target {
			t.Errorf("safe %q became %q", target, got)
		}
	}
	for _, target := range []string{"", "https://evil.test", "//evil.test", "/\\evil.test", "/%2f%2fevil.test", "/%5cevil.test", "/%255cevil.test", "/api/delete", "/activate?token=another", "/api/auth/tongji", "/login?redirect=https://evil.test", "/reset-password?token=secret", "/unknown", "/p/post/0", "/p/post/12/../13", "/%0apublish"} {
		if got := activationContinuation(target); got != "/settings?tab=profile" {
			t.Errorf("unsafe %q accepted as %q", target, got)
		}
	}
}

// Export real embedded SSR output only for the browser acceptance harness.
func TestActivationContinuationBrowserFixtures(t *testing.T) {
	dir := os.Getenv("YOURTJ_TEST_HTML_DIR")
	if dir == "" {
		t.Skip("browser output not requested")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"zh", "en", "ja", "de"} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("GET", "/activate?redirect=%2Fpublish%3Fid%3D42", nil)
		c.Request.Header.Set("Accept-Language", language)
		renderActivationSuccess(c, &users.EntityComplete{Username: "verified_student"})
		if err := os.WriteFile(filepath.Join(dir, language+".html"), rec.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
