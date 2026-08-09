package llmsservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/defaultconfig"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"gorm.io/gorm"
)

type llmsProjectionFixture struct {
	conn          *gorm.DB
	publicTopicID uint64
}

func TestLLMSProjectionVisibilityAndFeatureGates(t *testing.T) {
	fixture := setupLLMSProjectionFixture(t)
	host := "https://forum.example.test/ignored/path"

	setLLMSSettings(t, fixture.conn, pageConfig.LLMSConfig{Enabled: true})
	index, err := BuildIndex(host)
	if err != nil {
		t.Fatalf("BuildIndex() err=%v", err)
	}
	htmlURL := fmt.Sprintf("https://forum.example.test/p/post/%d", fixture.publicTopicID)
	assertContains(t, index, "# YourTJHub")
	assertContains(t, index, "## Topics")
	assertContains(t, index, htmlURL)
	assertContains(t, index, "Categories: Engineering")
	assertContains(t, index, "public excerpt")
	assertNotContains(t, index, "draft secret")
	assertNotContains(t, index, "blocked secret")
	assertNotContains(t, index, "blocked first post secret")

	if _, err := BuildFull(host); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("BuildFull() err=%v, want unavailable while fullText is disabled", err)
	}
	if _, err := BuildTopic(host, fixture.publicTopicID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("BuildTopic() err=%v, want unavailable while files is disabled", err)
	}

	setLLMSSettings(t, fixture.conn, pageConfig.LLMSConfig{Enabled: true, FullText: true, Files: true})
	index, err = BuildIndex(host)
	if err != nil {
		t.Fatalf("BuildIndex() with files err=%v", err)
	}
	markdownURL := fmt.Sprintf("https://forum.example.test/p/posts/%d.md", fixture.publicTopicID)
	assertContains(t, index, markdownURL)
	assertNotContains(t, index, htmlURL+")")

	full, err := BuildFull(host)
	if err != nil {
		t.Fatalf("BuildFull() err=%v", err)
	}
	assertContains(t, full, "## Public topic")
	assertContains(t, full, "**public body**")
	assertContains(t, full, "public reply")
	assertNotContains(t, full, "blocked reply secret")
	assertNotContains(t, full, "pending reply secret")
	assertNotContains(t, full, "deleted reply secret")
	assertNotContains(t, full, "draft secret")

	topic, err := BuildTopic(host, fixture.publicTopicID)
	if err != nil {
		t.Fatalf("BuildTopic() err=%v", err)
	}
	assertContains(t, topic, "# Public topic")
	assertContains(t, topic, "## Original post")
	assertContains(t, topic, "## Replies")
	assertContains(t, topic, "### Reply 2")
	if _, err := BuildTopic(host, fixture.publicTopicID+999); !errors.Is(err, ErrTopicMissing) {
		t.Fatalf("BuildTopic(missing) err=%v, want topic missing", err)
	}

	if err := fixture.conn.Model(&topics.Entity{}).
		Where("id = ?", fixture.publicTopicID).
		Update("excerpt", "updated excerpt").Error; err != nil {
		t.Fatalf("update cached topic excerpt: %v", err)
	}
	cached, err := BuildIndex(host)
	if err != nil {
		t.Fatalf("BuildIndex() cached err=%v", err)
	}
	assertNotContains(t, cached, "updated excerpt")
	ClearCache()
	refreshed, err := BuildIndex(host)
	if err != nil {
		t.Fatalf("BuildIndex() refreshed err=%v", err)
	}
	assertContains(t, refreshed, "updated excerpt")

	setLLMSSettings(t, fixture.conn, pageConfig.LLMSConfig{})
	if _, err := BuildIndex(host); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("BuildIndex() err=%v, want unavailable while master switch is disabled", err)
	}
}

func TestLLMSProjectionFormattingHelpers(t *testing.T) {
	if got := normalizeBaseURL("https://forum.example.test/path?q=1"); got != "https://forum.example.test" {
		t.Fatalf("normalizeBaseURL()=%q", got)
	}
	if got := normalizeBaseURL("javascript:alert(1)"); got != "" {
		t.Fatalf("normalizeBaseURL(javascript)=%q, want empty", got)
	}
	if got := escapeLinkLabel("topic [one] \\ test"); got != `topic \[one\] \\ test` {
		t.Fatalf("escapeLinkLabel()=%q", got)
	}
	if got := truncateRunes("同济大学", 2); got != "同济..." {
		t.Fatalf("truncateRunes()=%q", got)
	}
}

func setupLLMSProjectionFixture(t *testing.T) llmsProjectionFixture {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &category.Entity{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate llms projection tables: %v", err)
	}
	restoreLLMSPostingSettings(t, conn)

	base := uint64(time.Now().UnixNano()%1_000_000_000) + 6_800_000_000
	categoryID := base + 1
	publicTopicID := base + 10
	draftTopicID := base + 20
	blockedTopicID := base + 30
	blockedFirstPostTopicID := base + 40
	topicIDs := []uint64{publicTopicID, draftTopicID, blockedTopicID, blockedFirstPostTopicID}
	postIDs := []uint64{base + 101, base + 102, base + 103, base + 104, base + 105, base + 201, base + 301, base + 401}
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", postIDs).Delete(&posts.Entity{})
		conn.Unscoped().Where("id IN ?", topicIDs).Delete(&topics.Entity{})
		conn.Where("id = ?", categoryID).Delete(&category.Entity{})
		hotdataserve.ClearCategoryCache()
		ClearCache()
	})

	if err := conn.Create(&category.Entity{Id: categoryID, Name: "Engineering", Slug: "engineering"}).Error; err != nil {
		t.Fatalf("create llms category: %v", err)
	}
	hotdataserve.ClearCategoryCache()
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	postRows := []posts.Entity{
		{Id: postIDs[0], TopicId: publicTopicID, PostNo: 1, Content: "**public body**", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[1], TopicId: publicTopicID, PostNo: 2, Content: "public reply", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[2], TopicId: publicTopicID, PostNo: 3, Content: "blocked reply secret", ProcessStatus: posts.ProcessStatusBlocked, CreatedAt: now},
		{Id: postIDs[3], TopicId: publicTopicID, PostNo: 4, Content: "pending reply secret", ProcessStatus: posts.ProcessStatusPending, CreatedAt: now},
		{Id: postIDs[4], TopicId: publicTopicID, PostNo: 5, Content: "deleted reply secret", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[5], TopicId: draftTopicID, PostNo: 1, Content: "draft secret", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[6], TopicId: blockedTopicID, PostNo: 1, Content: "blocked secret", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[7], TopicId: blockedFirstPostTopicID, PostNo: 1, Content: "blocked first post secret", ProcessStatus: posts.ProcessStatusBlocked, CreatedAt: now},
	}
	if err := conn.Create(&postRows).Error; err != nil {
		t.Fatalf("create llms posts: %v", err)
	}
	if err := conn.Delete(&postRows[4]).Error; err != nil {
		t.Fatalf("soft delete llms reply: %v", err)
	}
	topicRows := []topics.Entity{
		{Id: publicTopicID, Title: "Public topic", CategoryIds: []uint64{categoryID}, FirstPostId: postIDs[0], Status: 1, ProcessStatus: topics.ProcessStatusNormal, Excerpt: "public excerpt", CreatedAt: now, UpdatedAt: now},
		{Id: draftTopicID, Title: "draft secret", FirstPostId: postIDs[5], Status: 0, ProcessStatus: topics.ProcessStatusNormal, Excerpt: "draft secret", CreatedAt: now, UpdatedAt: now},
		{Id: blockedTopicID, Title: "blocked secret", FirstPostId: postIDs[6], Status: 1, ProcessStatus: topics.ProcessStatusBlocked, Excerpt: "blocked secret", CreatedAt: now, UpdatedAt: now},
		{Id: blockedFirstPostTopicID, Title: "blocked first post secret", FirstPostId: postIDs[7], Status: 1, ProcessStatus: topics.ProcessStatusNormal, Excerpt: "blocked first post secret", CreatedAt: now, UpdatedAt: now},
	}
	for index := range topicRows {
		if err := conn.Create(&topicRows[index]).Error; err != nil {
			t.Fatalf("create llms topic %d: %v", topicRows[index].Id, err)
		}
	}
	return llmsProjectionFixture{conn: conn, publicTopicID: publicTopicID}
}

func restoreLLMSPostingSettings(t *testing.T, conn *gorm.DB) {
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
		ClearCache()
	})
}

func setLLMSSettings(t *testing.T, conn *gorm.DB, llms pageConfig.LLMSConfig) {
	t.Helper()
	config := defaultconfig.GetDefaultPostingSettingsConfig()
	config.LLMS = llms
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode posting settings: %v", err)
	}
	entity := pageConfig.Entity{PageType: pageConfig.PostingSettings, Config: string(encoded)}
	if err := conn.Where("page_type = ?", pageConfig.PostingSettings).Assign(entity).FirstOrCreate(&entity).Error; err != nil {
		t.Fatalf("save posting settings: %v", err)
	}
	hotdataserve.ClearPostingSettingsConfigCache()
	ClearCache()
}

func assertContains(t *testing.T, value string, fragment string) {
	t.Helper()
	if !strings.Contains(value, fragment) {
		t.Fatalf("output does not contain %q:\n%s", fragment, value)
	}
}

func assertNotContains(t *testing.T, value string, fragment string) {
	t.Helper()
	if strings.Contains(value, fragment) {
		t.Fatalf("output unexpectedly contains %q:\n%s", fragment, value)
	}
}
