package eventhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"gorm.io/gorm"
)

func TestLLMSHandlersInvalidateProjectionCache(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate llms handler tables: %v", err)
	}
	restoreLLMSHandlerSettings(t, conn)

	base := uint64(time.Now().UnixNano()%1_000_000_000) + 7_900_000_000
	topicID := base + 1
	postID := base + 2
	t.Cleanup(func() {
		conn.Unscoped().Delete(&posts.Entity{}, postID)
		conn.Unscoped().Delete(&topics.Entity{}, topicID)
		llmsservice.ClearCache()
	})

	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, Content: "handler body", CreatedAt: now}).Error; err != nil {
		t.Fatalf("create handler post: %v", err)
	}
	topic := topics.Entity{Id: topicID, Title: "Handler cache topic", FirstPostId: postID, Status: 1, ProcessStatus: topics.ProcessStatusNormal, Excerpt: "before invalidation", CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create handler topic: %v", err)
	}
	persistLLMSHandlerSettings(t, conn)

	host := "https://events.example.test"
	initial, err := llmsservice.BuildIndex(host)
	if err != nil {
		t.Fatalf("BuildIndex() initial err=%v", err)
	}
	if !strings.Contains(initial, "before invalidation") {
		t.Fatalf("initial projection missing fixture: %s", initial)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("excerpt", "after invalidation").Error; err != nil {
		t.Fatalf("update handler topic: %v", err)
	}
	cached, err := llmsservice.BuildIndex(host)
	if err != nil {
		t.Fatalf("BuildIndex() cached err=%v", err)
	}
	if strings.Contains(cached, "after invalidation") {
		t.Fatal("projection refreshed before an invalidation event")
	}

	if err := handleLLMSTopicUpdated(context.Background(), &TopicUpdatedEvent{Topic: &topic}); err != nil {
		t.Fatalf("handleLLMSTopicUpdated() err=%v", err)
	}
	refreshed, err := llmsservice.BuildIndex(host)
	if err != nil {
		t.Fatalf("BuildIndex() refreshed err=%v", err)
	}
	if !strings.Contains(refreshed, "after invalidation") {
		t.Fatalf("projection was not refreshed after event: %s", refreshed)
	}

	for name, invoke := range map[string]func() error{
		"published": func() error { return handleLLMSTopicPublished(context.Background(), &TopicPublishedEvent{}) },
		"deleted":   func() error { return handleLLMSTopicDeleted(context.Background(), &TopicDeletedEvent{}) },
		"comment":   func() error { return handleLLMSCommentCreated(context.Background(), &CommentCreatedEvent{}) },
		"category updated": func() error {
			return handleLLMSCategoryUpdated(context.Background(), &CategorySearchIndexUpdatedEvent{})
		},
		"category deleted": func() error {
			return handleLLMSCategoryDeleted(context.Background(), &CategorySearchIndexDeletedEvent{})
		},
	} {
		if err := invoke(); err != nil {
			t.Fatalf("%s llms handler err=%v", name, err)
		}
	}
}

func restoreLLMSHandlerSettings(t *testing.T, conn *gorm.DB) {
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

func persistLLMSHandlerSettings(t *testing.T, conn *gorm.DB) {
	t.Helper()
	config := defaultconfig.GetDefaultPostingSettingsConfig()
	config.LLMS.Enabled = true
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
