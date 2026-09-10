package forum

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"gorm.io/gorm"
)

// 审核封禁/解封主题与回复不发布事件，必须同步清 LLMS 缓存，
// 否则封禁内容在 10s 缓存窗口内仍可从公开投影导出。
func TestUpdateModerationTopicStatusInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSModerationCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_600_000_000
	modID := base + 1
	topicID := base + 2
	postID := base + 3
	createLLMSModerator(t, conn, modID)
	createLLMSModerationTopic(t, conn, topicID, postID, modID, "Moderated topic", "mod topic body")
	setLLMSModerationEnabled(t, conn)
	host := "https://modcache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "Moderated topic") {
		t.Fatalf("before projection missing topic: %s", before)
	}

	res := UpdateModerationTopicStatus(component.BetterRequest[ModerationTopicStatusReq]{
		UserId: modID,
		Params: ModerationTopicStatusReq{TopicId: topicID, Action: "ban"},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdateModerationTopicStatus() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if strings.Contains(after, "Moderated topic") {
		t.Fatalf("banned topic still exported after UpdateModerationTopicStatus:\n%s", after)
	}
}

func TestUpdateModerationPostStatusInvalidatesLLMSCache(t *testing.T) {
	conn := setupLLMSModerationCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_601_000_000
	modID := base + 1
	topicID := base + 2
	post1ID := base + 3
	post2ID := base + 4
	createLLMSModerator(t, conn, modID)
	createLLMSModerationTopic(t, conn, topicID, post1ID, modID, "Moderated reply topic", "first body")
	createLLMSModerationReply(t, conn, post2ID, topicID, modID, 2, "moderated reply body")
	setLLMSModerationEnabled(t, conn)
	host := "https://modcache.example.test"

	before, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() before err=%v", err)
	}
	if !strings.Contains(before, "moderated reply body") {
		t.Fatalf("before projection missing reply: %s", before)
	}

	res := UpdateModerationPostStatus(component.BetterRequest[ModerationPostStatusReq]{
		UserId: modID,
		Params: ModerationPostStatusReq{PostId: post2ID, Action: "ban"},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdateModerationPostStatus() failed: %+v", res)
	}

	after, err := llmsservice.BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() after err=%v", err)
	}
	if strings.Contains(after, "moderated reply body") {
		t.Fatalf("banned reply still exported after UpdateModerationPostStatus:\n%s", after)
	}
}

func setupLLMSModerationCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	err := conn.AutoMigrate(
		&pageConfig.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&moderators.Entity{},
		&moderationLog.Entity{},
		&users.EntityComplete{},
		&taskQueue.Entity{},
	)
	if err != nil {
		t.Fatalf("migrate llms moderation cache tables: %v", err)
	}
	restoreLLMSModerationSettings(t, conn)
	return conn
}

func setLLMSModerationEnabled(t *testing.T, conn *gorm.DB) {
	t.Helper()
	config := defaultconfig.GetDefaultPostingSettingsConfig()
	config.LLMS = pageConfig.LLMSConfig{Enabled: true, FullText: true}
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

func restoreLLMSModerationSettings(t *testing.T, conn *gorm.DB) {
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

func createLLMSModerator(t *testing.T, conn *gorm.DB, userID uint64) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("user_id = ?", userID).Delete(&moderators.Entity{})
		conn.Unscoped().Where("id = ?", userID).Delete(&users.EntityComplete{})
		moderationservice.Invalidate()
	})
	now := time.Now().Add(-time.Hour)
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: "llms_moderator", IsActivated: 1, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create moderator user: %v", err)
	}
	if err := conn.Create(&moderators.Entity{UserId: userID, ScopeType: moderators.ScopeGlobal, Status: 1}).Error; err != nil {
		t.Fatalf("create moderator grant: %v", err)
	}
	moderationservice.Invalidate()
}

func createLLMSModerationTopic(t *testing.T, conn *gorm.DB, topicID, postID, userID uint64, title, content string) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", postID).Delete(&posts.Entity{})
		conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
		llmsservice.ClearCache()
	})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, UserId: userID, Content: content, ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create llms moderation first post: %v", err)
	}
	if err := conn.Create(&topics.Entity{Id: topicID, Title: title, FirstPostId: postID, Status: 1, ProcessStatus: topics.ProcessStatusNormal, UserId: userID, Excerpt: title + " excerpt", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create llms moderation topic: %v", err)
	}
}

func createLLMSModerationReply(t *testing.T, conn *gorm.DB, postID, topicID, userID uint64, postNo uint64, content string) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", postID).Delete(&posts.Entity{})
		llmsservice.ClearCache()
	})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: postNo, UserId: userID, Content: content, ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create llms moderation reply: %v", err)
	}
}
