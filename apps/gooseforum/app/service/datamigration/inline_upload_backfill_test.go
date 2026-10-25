package datamigration

import (
	"encoding/json"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func TestBackfillProtectsInlineUploadsWithoutPublishingPrivateImages(t *testing.T) {
	conn := prepareChatCoverBackfillDB(t)
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}); err != nil {
		t.Fatal(err)
	}
	const owner = 9800500901
	const prefix = "https://cdn.example.test/media"
	config, err := json.Marshal(pageConfig.StorageSettingsStorage{Provider: "local", PublicUrlPrefix: prefix})
	if err != nil {
		t.Fatal(err)
	}
	setting := pageConfig.Entity{PageType: pageConfig.StorageSettingsPage, Config: string(config)}
	if err := conn.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	hotdataserve.ClearStorageSettingsConfigCache()
	t.Cleanup(func() { conn.Delete(&setting); hotdataserve.ClearStorageSettingsConfigCache() })
	names := []string{"2026/10/old-cdn-gallery.webp", "2026/10/old-cdn-post.webp", "2026/10/old-cdn-revision.webp", "2026/10/old-cdn-deleted.webp", "2026/10/old-cdn-foreign.webp", "2026/10/old-cdn-deleted-gallery.webp"}
	for i, name := range names {
		uploader := uint64(owner)
		if i == 4 {
			uploader++
		}
		seedLegacyUpload(t, uploader, name)
	}
	topic := topics.Entity{Id: 9800500902, UserId: owner, Status: 1, FirstPostId: 9800500903, ImageUrls: []string{prefix + "/" + names[0]}}
	post := posts.Entity{Id: 9800500903, UserId: owner, TopicId: topic.Id, PostNo: 1, Content: "![](" + prefix + "/" + names[1] + ") ![](" + prefix + "/" + names[4] + ")"}
	privateTopic := topics.Entity{Id: 9800500906, UserId: owner, Status: 1, FirstPostId: 9800500904, ImageUrls: []string{prefix + "/" + names[5]}}
	deleted := posts.Entity{Id: 9800500904, UserId: owner, TopicId: privateTopic.Id, PostNo: 1, Content: "![](" + prefix + "/" + names[3] + ")", VisibilityStatus: posts.VisibilityUserDeleted}
	revision := postRevisions.Entity{Id: 9800500905, PostId: post.Id, Version: 1, Content: "![](" + prefix + "/" + names[2] + ")", ProcessStatus: posts.ProcessStatusPending}
	for _, value := range []any{&topic, &privateTopic, &post, &deleted, &revision} {
		if err := conn.Create(value).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(value) })
	}
	result := BackfillFileUsagesChatCoverWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("backfill: %+v", result)
	}
	for i, name := range names {
		var refs []fileUsage.Entity
		if err := conn.Where("file_name = ?", name).Find(&refs).Error; err != nil {
			t.Fatal(err)
		}
		var found bool
		for _, ref := range refs {
			if ref.UsageType != fileUsage.UsageUploadOwner {
				found = true
				if i < 2 && ref.Status != fileUsage.UsageStatusActive {
					t.Fatalf("public image not restored: %+v", ref)
				}
				if i >= 2 && ref.Status == fileUsage.UsageStatusActive {
					t.Fatalf("private/foreign file made public: %+v", ref)
				}
			}
		}
		if found != (i != 4) {
			t.Errorf("reference for %s = %v", name, found)
		}
	}
	for i, name := range names {
		if i == 4 {
			continue
		}
		// A main-DB claim is the authoritative prerequisite to byte deletion.
		if claimed, err := fileUsage.ClaimOrphanUpload(name); err != nil || claimed {
			t.Errorf("retained content was reclaimable: %s %v %v", name, claimed, err)
		}
	}
	again := BackfillFileUsagesChatCoverWithDB(conn)
	if again.Failed != 0 || again.Inserted != 0 {
		t.Errorf("not idempotent: %+v", again)
	}
}
