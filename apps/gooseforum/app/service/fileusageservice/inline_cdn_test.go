package fileusageservice

import (
	"encoding/json"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func TestInlineCDNUploadRemainsReferenced(t *testing.T) {
	cleanupTestUsageRows(t)
	const owner = 9800300991
	const name = "2026/10/published-cdn-inline.webp"
	seedOrphanCandidate(t, owner, name, time.Now().Add(-8*24*time.Hour))
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	config, err := json.Marshal(pageConfig.StorageSettingsStorage{Provider: "local", PublicUrlPrefix: "https://cdn.example.test/media"})
	if err != nil {
		t.Fatal(err)
	}
	row := pageConfig.Entity{PageType: pageConfig.StorageSettingsPage, Config: string(config)}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	hotdataserve.ClearStorageSettingsConfigCache()
	t.Cleanup(func() { conn.Delete(&row); hotdataserve.ClearStorageSettingsConfigCache() })
	url := "https://cdn.example.test/media/" + name
	if got := GalleryImageURLs([]string{url}); len(got) != 1 {
		t.Errorf("CDN gallery dropped: %v", got)
	}
	images, err := ResolveOwnedInlineImages(owner, "![image]("+url+")", nil)
	if err != nil || len(images) != 1 {
		t.Fatalf("published CDN image has no reference: %v %v", images, err)
	}
	if err := RegisterTopicInlineImagesTx(conn, 9800300992, owner, images, false); err != nil {
		t.Fatal(err)
	}
	if removed, err := pruneOrphanUpload(name); err != nil || removed {
		t.Fatalf("published CDN image reclaimed: %v %v", removed, err)
	}
}
