package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/gin-gonic/gin"
)

func getImageWithHeaders(router http.Handler, path, token string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestFileImageETagAndNotModified(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	owner := createHTTPContractUser(t, conn, contractTestID())
	ownerToken := contractSessionToken(t, owner)
	image, name := saveContractImage(t, owner.Id)
	if err := fileusageservice.AddUploadOwner(owner.Id, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Where("file_name = ?", name).Delete(&fileUsage.Entity{}) })

	// Private orphan upload: owner preview never exposes ETag and ignores If-None-Match.
	priv := getImageWithHeaders(router, image, ownerToken, map[string]string{"If-None-Match": "*"})
	if priv.Code != http.StatusOK || priv.Header().Get("Cache-Control") != "private, no-store" || priv.Header().Get("ETag") != "" {
		t.Fatalf("private preview = %d cache=%q etag=%q, want 200 private, no-store without ETag",
			priv.Code, priv.Header().Get("Cache-Control"), priv.Header().Get("ETag"))
	}

	// Publish the image into a topic => public, max-age=300 + ETag.
	ref := fileUsage.Entity{
		FileName:   name,
		TargetType: fileUsage.TargetTopic,
		TargetId:   contractTestID(),
		UserId:     owner.Id,
		UsageType:  fileUsage.UsageInlineImage,
		Status:     fileUsage.UsageStatusActive,
	}
	if err := conn.Create(&ref).Error; err != nil {
		t.Fatal(err)
	}
	first := getImage(router, image, "")
	if first.Code != http.StatusOK {
		t.Fatalf("first public read = %d, want 200", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" || first.Header().Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("first public headers: Cache-Control=%q ETag=%q", first.Header().Get("Cache-Control"), etag)
	}

	// Strong match, weak match, and comma-separated list all return 304 without body.
	for _, header := range []string{etag, "W/" + etag, `"other", ` + etag} {
		reval := getImageWithHeaders(router, image, "", map[string]string{"If-None-Match": header})
		if reval.Code != http.StatusNotModified || reval.Body.Len() != 0 {
			t.Errorf("If-None-Match=%s: status=%d bodyLen=%d, want 304 empty", header, reval.Code, reval.Body.Len())
		}
		if reval.Header().Get("Cache-Control") != "public, max-age=300" || reval.Header().Get("ETag") != etag {
			t.Errorf("If-None-Match=%s: Cache-Control=%q ETag=%q", header, reval.Header().Get("Cache-Control"), reval.Header().Get("ETag"))
		}
	}

	// Mismatched ETag returns full 200.
	miss := getImageWithHeaders(router, image, "", map[string]string{"If-None-Match": `"f-0-0-0"`})
	if miss.Code != http.StatusOK || miss.Body.Len() == 0 {
		t.Fatalf("mismatched ETag: status=%d bodyLen=%d, want 200 with body", miss.Code, miss.Body.Len())
	}

	// Once withdrawn, revalidation with the old ETag must fail with 404 private, no-store.
	if err := conn.Model(&ref).Update("status", fileUsage.UsageStatusRecovering).Error; err != nil {
		t.Fatal(err)
	}
	revoked := getImageWithHeaders(router, image, "", map[string]string{"If-None-Match": etag})
	if revoked.Code != http.StatusNotFound || revoked.Header().Get("Cache-Control") != "private, no-store" || revoked.Header().Get("ETag") != "" {
		t.Fatalf("withdrawn revalidation: status=%d cache=%q etag=%q", revoked.Code, revoked.Header().Get("Cache-Control"), revoked.Header().Get("ETag"))
	}
}

func TestStaticWebPExcludedFromGzipAndVary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldEnv := preferences.GetString("app.env", "production")
	oldGzip := preferences.GetBool("server.gzip", true)
	preferences.Set("app.env", "production")
	preferences.Set("server.gzip", true)
	t.Cleanup(func() {
		preferences.Set("app.env", oldEnv)
		preferences.Set("server.gzip", oldGzip)
	})

	router := gin.New()
	assertRouter(router)

	req := httptest.NewRequest(http.MethodGet, "/static/pic/default-avatar.webp", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/pic/default-avatar.webp status = %d, want 200", rec.Code)
	}
	if ce := rec.Header().Get("Content-Encoding"); ce != "" {
		t.Errorf("webp Content-Encoding = %q, want empty (already compressed)", ce)
	}
	if vary := rec.Header().Get("Vary"); vary != "" {
		t.Errorf("webp Vary = %q, want empty", vary)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=18144000" {
		t.Errorf("webp Cache-Control = %q, want public, max-age=18144000", cc)
	}
}
