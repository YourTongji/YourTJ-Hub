package routes

import (
	"encoding/json"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
)

func TestOrphanUploadIsPrivateUntilPublished(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	image, name := saveContractImage(t, owner.Id)
	if err := fileusageservice.AddUploadOwner(owner.Id, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Where("file_name = ?", name).Delete(&fileUsage.Entity{}) })
	original := filedata.GetByName(name)
	variant, err := filedata.SaveFile(owner.Id, name+"-small.png", "image/png", original.Data)
	if err != nil {
		t.Fatal(err)
	}
	if err := db4fileconnect.Connect().Model(variant).UpdateColumn("parent_name", name).Error; err != nil {
		t.Fatal(err)
	}
	images := []string{image, variant.GetAccessPath()}
	ownerToken := contractSessionToken(t, owner)
	otherToken := contractSessionToken(t, other)
	for _, tc := range []struct {
		label, token string
		status       int
	}{
		{"anonymous", "", http.StatusNotFound},
		{"other user", otherToken, http.StatusNotFound},
		{"owner", ownerToken, http.StatusOK},
	} {
		for _, image := range images {
			res := getImage(router, image, tc.token)
			if res.Code != tc.status {
				t.Errorf("%s: status=%d, want %d", tc.label, res.Code, tc.status)
			}
			if res.Header().Get("Cache-Control") != "private, no-store" {
				t.Errorf("%s: private response is cacheable", tc.label)
			}
		}
	}
	ref := fileUsage.Entity{FileName: name, TargetType: fileUsage.TargetTopic, TargetId: contractTestID(), UserId: owner.Id, UsageType: fileUsage.UsageInlineImage, Status: fileUsage.UsageStatusActive}
	if err := conn.Create(&ref).Error; err != nil {
		t.Fatal(err)
	}
	res := getImage(router, image, "")
	if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("public response: status=%d cache=%q", res.Code, res.Header().Get("Cache-Control"))
	}
	expired := time.Now().Add(-time.Hour)
	if err := conn.Model(&ref).Update("expires_at", expired).Error; err != nil {
		t.Fatal(err)
	}
	if res := getImage(router, image, ""); res.Code != http.StatusNotFound {
		t.Fatalf("expired reference remained public: %d", res.Code)
	}
	if err := conn.Model(&ref).Update("status", fileUsage.UsageStatusRecovering).Error; err != nil {
		t.Fatal(err)
	}
	if res := getImage(router, image, ownerToken); res.Code != http.StatusNotFound {
		t.Fatalf("owner can still read withdrawn content: %d", res.Code)
	}
}

func TestLegacyUntrackedImageRemainsReadableWithoutLongCache(t *testing.T) {
	_, router, _ := setupAIModerationContractTest(t, nil)
	image, _ := saveContractImage(t, 0)
	res := getImage(router, image, "")
	if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("legacy response: status=%d cache=%q", res.Code, res.Header().Get("Cache-Control"))
	}
}

func TestImageReferenceLookupFailureDeniesRead(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	image, _ := saveContractImage(t, 0)
	if err := conn.Migrator().RenameTable("file_usages", "hotfix_file_usages_unavailable"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Migrator().RenameTable("hotfix_file_usages_unavailable", "file_usages"); err != nil {
			t.Fatal(err)
		}
	}()
	res := getImage(router, image, "")
	if res.Code != http.StatusServiceUnavailable || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("lookup failure: status=%d cache=%q", res.Code, res.Header().Get("Cache-Control"))
	}
}

func TestMultipartUploadRemainsPrivateBeforePublication(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	router.POST("/file/img-upload", middleware.JWTAuthCheck, api.SaveImgByGinContext)
	owner := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	response := serveMultipartNamed(router, "/file/img-upload", map[string]routeNamedFile{"file": {name: "orphan.png", data: contractTinyPNG}}, token)
	envelope := decodeContractEnvelope(t, response)
	if response.Code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("upload status=%d envelope=%+v", response.Code, envelope)
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.URL == "" {
		t.Fatal("upload returned no URL")
	}
	name := fileusageservice.FileNameFromURL(result.URL)
	t.Cleanup(func() { _ = filedata.DeleteByName(name); conn.Where("file_name = ?", name).Delete(&fileUsage.Entity{}) })
	if res := getImage(router, result.URL, ""); res.Code != http.StatusNotFound {
		t.Fatalf("unpublished multipart upload is public: %d", res.Code)
	}
	if res := getImage(router, result.URL, token); res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("private multipart preview status=%d", res.Code)
	}
}

func TestMultipartUploadRollsBackOnOwnershipFailure(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	router.POST("/file/img-upload", middleware.JWTAuthCheck, api.SaveImgByGinContext)
	owner := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	if err := conn.Migrator().RenameTable("file_usages", "hotfix_file_usages_unavailable"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Migrator().RenameTable("hotfix_file_usages_unavailable", "file_usages"); err != nil {
			t.Fatal(err)
		}
	}()
	response := serveMultipartNamed(router, "/file/img-upload", map[string]routeNamedFile{"file": {name: "orphan.png", data: contractTinyPNG}}, token)
	envelope := decodeContractEnvelope(t, response)
	if response.Code != http.StatusInternalServerError || envelope.MessageCode != "upload.saveFailed" {
		t.Fatalf("ownership failure status=%d code=%q", response.Code, envelope.MessageCode)
	}
	var count int64
	if err := db4fileconnect.Connect().Model(&filedata.Entity{}).Where("user_id = ?", owner.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("ownership failure left %d uploaded files", count)
	}
}
