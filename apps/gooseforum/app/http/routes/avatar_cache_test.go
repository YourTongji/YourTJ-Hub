package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
)

func TestAvatarFileCacheHeader(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	image, name := saveContractImage(t, 0)
	original := filedata.GetByName(name)
	variant, err := filedata.SaveFile(0, name+"-small.png", "image/png", original.Data)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = filedata.DeleteByName(variant.Name) })
	if err := db4fileconnect.Connect().Model(variant).UpdateColumn("parent_name", name).Error; err != nil {
		t.Fatal(err)
	}
	ref := fileUsage.Entity{FileName: name, TargetType: fileUsage.TargetUser, TargetId: contractTestID(), UsageType: fileUsage.UsageAvatar, Status: fileUsage.UsageStatusActive}
	if err := conn.Create(&ref).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&ref) })
	expired := time.Now().Add(-time.Minute)
	for _, tc := range []struct {
		name, status, cache string
		expires             *time.Time
		code                int
	}{
		{"active", fileUsage.UsageStatusActive, "public, max-age=300", nil, http.StatusOK},
		{"expired", fileUsage.UsageStatusActive, "private, no-store", &expired, http.StatusNotFound},
		{"pending", fileUsage.UsageStatusPending, "private, no-store", nil, http.StatusNotFound},
		{"withdrawn", fileUsage.UsageStatusRecovering, "private, no-store", nil, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := conn.Model(&ref).Updates(map[string]any{"status": tc.status, "expires_at": tc.expires}).Error; err != nil {
				t.Fatal(err)
			}
			for _, url := range []string{image, variant.GetAccessPath()} {
				res := getImage(router, url, "")
				if res.Code != tc.code || res.Header().Get("Cache-Control") != tc.cache {
					t.Errorf("%s: status=%d cache=%q, want %d %q", url, res.Code, res.Header().Get("Cache-Control"), tc.code, tc.cache)
				}
			}
		})
	}
}
