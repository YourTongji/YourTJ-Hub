package api

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/defaultconfig"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topicCategoryIndex"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/llmsservice"
	"gorm.io/gorm"
)

// 这些路径会变更 llms 投影可见的数据（封禁/改删回复/改分类/下架），
// 必须同步调用 llmsservice.ClearCache()，否则 10s 缓存窗口内公开投影仍会泄露陈旧内容。
func TestEditTopicInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_400_000_000
	topicID := base + 1
	postID := base + 2
	createLLMSCacheTopic(t, conn, topicID, postID, base, "Cache topic A", "cache body A", nil)
	setLLMSCacheEnabled(t, conn)
	host := "https://cache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "Cache topic A") {
		t.Fatalf("before projection missing topic: %s", before)
	}

	res := EditTopic(component.BetterRequest[EditTopicReq]{
		UserId: base,
		Params: EditTopicReq{TopicId: topicID, ProcessStatus: topics.ProcessStatusBlocked},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("EditTopic() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if strings.Contains(after, "Cache topic A") {
		t.Fatalf("blocked topic still exported after EditTopic:\n%s", after)
	}
}

func TestEditTopicCategoriesInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_401_000_000
	oldCat := base + 1
	newCat := base + 2
	topicID := base + 3
	postID := base + 4
	createLLMSCacheCategory(t, conn, oldCat, "Old Cat", "old")
	createLLMSCacheCategory(t, conn, newCat, "New Cat", "new")
	createLLMSCacheTopic(t, conn, topicID, postID, base, "Category topic", "cat body", []uint64{oldCat})
	setLLMSCacheEnabled(t, conn)
	host := "https://cache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "Categories: Old Cat") {
		t.Fatalf("before projection missing old category: %s", before)
	}

	res := EditTopicCategories(component.BetterRequest[EditTopicCategoriesReq]{
		UserId: base,
		Params: EditTopicCategoriesReq{TopicId: topicID, CategoryId: []uint64{newCat}},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("EditTopicCategories() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if !strings.Contains(after, "Categories: New Cat") {
		t.Fatalf("after projection missing new category:\n%s", after)
	}
	if strings.Contains(after, "Categories: Old Cat") {
		t.Fatalf("old category still exported after EditTopicCategories:\n%s", after)
	}
}

func TestUpdatePostInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_402_000_000
	userID := base + 1
	topicID := base + 2
	post1ID := base + 3
	post2ID := base + 4
	createLLMSCacheTopic(t, conn, topicID, post1ID, userID, "Reply topic", "first body", nil)
	createLLMSCacheReply(t, conn, post2ID, topicID, userID, 2, "old reply content")
	setLLMSCacheEnabled(t, conn)
	host := "https://cache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "old reply content") {
		t.Fatalf("before projection missing reply: %s", before)
	}

	res := UpdatePost(component.BetterRequest[UpdatePostReq]{
		UserId: userID,
		Params: UpdatePostReq{PostId: post2ID, Content: "brand new reply content"},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdatePost() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if !strings.Contains(after, "brand new reply content") {
		t.Fatalf("after projection missing updated reply:\n%s", after)
	}
	if strings.Contains(after, "old reply content") {
		t.Fatalf("stale reply still exported after UpdatePost:\n%s", after)
	}
}

func TestDeletePostInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_403_000_000
	userID := base + 1
	topicID := base + 2
	post1ID := base + 3
	post2ID := base + 4
	createLLMSCacheTopic(t, conn, topicID, post1ID, userID, "Delete topic", "delete body", nil)
	createLLMSCacheReply(t, conn, post2ID, topicID, userID, 2, "reply to delete")
	setLLMSCacheEnabled(t, conn)
	host := "https://cache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "reply to delete") {
		t.Fatalf("before projection missing reply: %s", before)
	}

	res := DeletePost(component.BetterRequest[DeletePostReq]{
		UserId: userID,
		Params: DeletePostReq{PostId: post2ID},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("DeletePost() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if strings.Contains(after, "reply to delete") {
		t.Fatalf("deleted reply still exported after DeletePost:\n%s", after)
	}
}

func TestUpdateTopicStatusInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_404_000_000
	ownerID := base + 1
	topicID := base + 2
	postID := base + 3
	createLLMSCacheTopic(t, conn, topicID, postID, ownerID, "Unpublish topic", "unpublish body", nil)
	setLLMSCacheEnabled(t, conn)
	host := "https://cache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "Unpublish topic") {
		t.Fatalf("before projection missing topic: %s", before)
	}

	res := UpdateTopicStatus(component.BetterRequest[TopicStatusReq]{
		UserId: ownerID,
		Params: TopicStatusReq{TopicId: topicID, TopicStatus: 0},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdateTopicStatus() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if strings.Contains(after, "Unpublish topic") {
		t.Fatalf("unpublished topic still exported after UpdateTopicStatus:\n%s", after)
	}
}

func TestWriteTopicUnpublishInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_405_000_000
	userID := base + 1
	catID := base + 2
	topicID := base + 3
	postID := base + 4
	createLLMSCacheUser(t, conn, userID)
	createLLMSCacheCategory(t, conn, catID, "Publish Cat", "publish-cat")
	createLLMSCacheTopic(t, conn, topicID, postID, userID, "Publish topic", "publish body", []uint64{catID})
	setLLMSCacheEnabled(t, conn)
	host := "https://cache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "Publish topic") {
		t.Fatalf("before projection missing topic: %s", before)
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: userID,
		Params: WriteTopicReq{
			TopicId:     topicID,
			Title:       "Publish topic",
			Content:     "publish body after unpublish edit",
			CategoryId:  []uint64{catID},
			TopicStatus: 0,
		},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("WriteTopic() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if strings.Contains(after, "Publish topic") {
		t.Fatalf("unpublished topic still exported after WriteTopic edit:\n%s", after)
	}
}

func setupLLMSCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	err := conn.AutoMigrate(
		&pageConfig.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&pointsRecord.Entity{},
		&userPoints.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&users.EntityComplete{},
		&fileUsage.Entity{},
	)
	if err != nil {
		t.Fatalf("migrate llms cache tables: %v", err)
	}
	restoreLLMSCacheSettings(t, conn)
	return conn
}

func setLLMSCacheEnabled(t *testing.T, conn *gorm.DB) {
	t.Helper()
	config := defaultconfig.GetDefaultPostingSettingsConfig()
	config.LLMS = pageConfig.LLMSConfig{Enabled: true, FullText: true, Files: true}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode posting settings: %v", err)
	}
	entity := pageConfig.Entity{PageType: pageConfig.PostingSettings, Config: string(encoded)}
	if err := conn.Where("page_type = ?", pageConfig.PostingSettings).Assign(entity).FirstOrCreate(&entity).Error; err != nil {
		t.Fatalf("save posting settings: %v", err)
	}
	hotdataserve.ClearPostingSettingsConfigCache()
	llmsservice.ClearCache()
}

func restoreLLMSCacheSettings(t *testing.T, conn *gorm.DB) {
	t.Helper()
	var previous pageConfig.Entity
	result := conn.Where("page_type = ?", pageConfig.PostingSettings).First(&previous)
	hadPrevious := result.Error == nil
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		t.Fatalf("read existing posting settings: %v", result.Error)
	}
	t.Cleanup(func() {
		if hadPrevious {
			if err := conn.Save(&previous).Error; err != nil {
				t.Errorf("restore posting settings: %v", err)
			}
		} else if err := conn.Where("page_type = ?", pageConfig.PostingSettings).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("delete posting settings fixture: %v", err)
		}
		hotdataserve.ClearPostingSettingsConfigCache()
		llmsservice.ClearCache()
	})
}

func createLLMSCacheTopic(t *testing.T, conn *gorm.DB, topicID, postID, userID uint64, title, content string, categoryIDs []uint64) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", postID).Delete(&posts.Entity{})
		conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
		llmsservice.ClearCache()
	})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, UserId: userID, Content: content, ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create llms cache first post: %v", err)
	}
	if err := conn.Create(&topics.Entity{Id: topicID, Title: title, CategoryIds: categoryIDs, FirstPostId: postID, Status: 1, ProcessStatus: topics.ProcessStatusNormal, UserId: userID, Excerpt: title + " excerpt", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create llms cache topic: %v", err)
	}
}

func createLLMSCacheReply(t *testing.T, conn *gorm.DB, postID, topicID, userID uint64, postNo uint64, content string) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", postID).Delete(&posts.Entity{})
		llmsservice.ClearCache()
	})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: postNo, UserId: userID, Content: content, ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create llms cache reply: %v", err)
	}
}

func createLLMSCacheCategory(t *testing.T, conn *gorm.DB, id uint64, name, slug string) {
	t.Helper()
	t.Cleanup(func() {
		conn.Where("id = ?", id).Delete(&category.Entity{})
		hotdataserve.ClearCategoryCache()
	})
	if err := conn.Create(&category.Entity{Id: id, Name: name, Slug: slug}).Error; err != nil {
		t.Fatalf("create llms cache category: %v", err)
	}
	hotdataserve.ClearCategoryCache()
}

func createLLMSCacheUser(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", id).Delete(&users.EntityComplete{})
	})
	now := time.Now().Add(-time.Hour)
	if err := conn.Create(&users.EntityComplete{Id: id, Username: "llms_cache_user", IsActivated: 1, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create llms cache user: %v", err)
	}
}
