package routes

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

// Issue #1228: the topic gallery (topics.image_urls) reaches Agents on the
// visible first post and on the topic list item.

var galleryFixture = []string{"/file/img/2026/09/09/a.webp", "https://cdn.example.test/b.webp"}

func createGalleryTopic(t *testing.T, conn *gorm.DB, topicID, firstPostID, authorID uint64, firstPost posts.Entity) {
	t.Helper()
	conn.Unscoped().Where("id IN ? OR topic_id = ?", []uint64{firstPostID, firstPostID + 1}, topicID).Delete(&posts.Entity{})
	conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
	now := time.Now().Add(-time.Hour)
	firstPost.Id, firstPost.TopicId, firstPost.PostNo, firstPost.UserId = firstPostID, topicID, 1, authorID
	firstPost.CreatedAt, firstPost.UpdatedAt = now, now
	if firstPost.Content == "" {
		firstPost.Content = "见图三"
	}
	if firstPost.VisibilityStatus == "" {
		firstPost.VisibilityStatus = posts.VisibilityActive
	}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	reply := posts.Entity{Id: firstPostID + 1, TopicId: topicID, PostNo: 2, UserId: authorID, Content: "reply", ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityActive, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&reply).Error; err != nil {
		t.Fatalf("create reply: %v", err)
	}
	topic := topics.Entity{Id: topicID, Title: "Gallery topic", UserId: authorID, Status: 1, ProcessStatus: 0, FirstPostId: firstPostID, LastPostId: reply.Id, PostCount: 2, PostSeq: 2, CategoryIds: []uint64{5101}, ImageUrls: galleryFixture, VisibilityStatus: topics.VisibilityActive, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
}

type galleryPost struct {
	PostNo uint64   `json:"postNo"`
	Images []string `json:"images"`
}

func agentGalleryWindow(t *testing.T, token string, topicID string) []galleryPost {
	t.Helper()
	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/topics/"+topicID+"/posts", "", token)
	if rec.Code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("post window failed: %s", rec.Body.String())
	}
	var window struct {
		Posts []galleryPost `json:"posts"`
	}
	if err := json.Unmarshal(envelope.Result, &window); err != nil {
		t.Fatalf("decode window %s: %v", envelope.Result, err)
	}
	return window.Posts
}

func TestAgentPostWindowFirstPostCarriesGallery(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "gallery-agent")
	createAgentForumCategory(t, conn, 5101, "gallery")
	createGalleryTopic(t, conn, 6101, 11010, agentID+1000, posts.Entity{ProcessStatus: posts.ProcessStatusNormal})

	items := agentGalleryWindow(t, token, "6101")
	if len(items) != 2 {
		t.Fatalf("posts = %#v, want first post and reply", items)
	}
	for _, item := range items {
		switch item.PostNo {
		case 1:
			if !reflect.DeepEqual(item.Images, galleryFixture) {
				t.Fatalf("first post images = %#v, want %#v", item.Images, galleryFixture)
			}
		default:
			if item.Images != nil {
				t.Fatalf("reply %d images = %#v, want omitted", item.PostNo, item.Images)
			}
		}
	}
}

func TestAgentPostWindowOmitsGalleryForUnavailableFirstPost(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first posts.Entity
	}{
		{"hidden", posts.Entity{ProcessStatus: 1}},
		{"author deleted", posts.Entity{ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityUserDeleted}},
		{"moderator removed", posts.Entity{ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityModeratorRemoved}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := setupAgentForumTestDB(t)
			agentID, token := createAgentForumAgent(t, conn, "gallery-hidden-"+strings.ReplaceAll(tc.name, " ", "-"))
			createAgentForumCategory(t, conn, 5101, "gallery")
			createGalleryTopic(t, conn, 6102, 11008, agentID+1000, tc.first)
			items := agentGalleryWindow(t, token, "6102")
			sawFirst := false
			for _, item := range items {
				sawFirst = sawFirst || item.PostNo == 1
				if item.Images != nil {
					t.Fatalf("post %d images = %#v, want omitted", item.PostNo, item.Images)
				}
			}
			if !sawFirst {
				t.Fatalf("first post placeholder missing from window: %#v", items)
			}
		})
	}
}

func TestAgentPostWindowInvisibleTopicStillNotFound(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "gallery-invisible-agent")
	createAgentForumCategory(t, conn, 5101, "gallery")
	createGalleryTopic(t, conn, 6103, 11006, agentID+1000, posts.Entity{ProcessStatus: posts.ProcessStatusNormal})
	if err := conn.Model(&topics.Entity{}).Where("id = ?", 6103).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/topics/6103/posts", "", token)
	if rec.Code != http.StatusOK || envelope.Code == 0 {
		t.Fatalf("invisible topic should be a business not-found, got %s", rec.Body.String())
	}
	if string(envelope.Result) != "" && string(envelope.Result) != "null" {
		t.Fatalf("invisible topic leaked result: %s", envelope.Result)
	}
}

func TestAgentTopicListCarriesGallery(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "gallery-list-agent")
	createAgentForumCategory(t, conn, 5101, "gallery")
	createGalleryTopic(t, conn, 6104, 11004, agentID+1000, posts.Entity{ProcessStatus: posts.ProcessStatusNormal})
	conn.Unscoped().Where("id = ?", 6105).Delete(&topics.Entity{})
	conn.Unscoped().Where("id = ? OR topic_id = ?", 11012, 6105).Delete(&posts.Entity{})
	plainPost := posts.Entity{Id: 11012, TopicId: 6105, PostNo: 1, UserId: agentID + 1000, Content: "plain", ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityActive, CreatedAt: time.Now().Add(-2 * time.Hour), UpdatedAt: time.Now().Add(-2 * time.Hour)}
	if err := conn.Create(&plainPost).Error; err != nil {
		t.Fatal(err)
	}
	plain := topics.Entity{Id: 6105, Title: "No gallery", UserId: agentID + 1000, Status: 1, FirstPostId: 11012, VisibilityStatus: topics.VisibilityActive, CategoryIds: []uint64{5101}, CreatedAt: time.Now().Add(-2 * time.Hour), UpdatedAt: time.Now().Add(-2 * time.Hour)}
	if err := conn.Create(&plain).Error; err != nil {
		t.Fatal(err)
	}
	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/topics", "", token)
	if rec.Code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("topics failed: %s", rec.Body.String())
	}
	var list struct {
		List []struct {
			Id     uint64    `json:"id"`
			Images *[]string `json:"images"`
		} `json:"list"`
	}
	if err := json.Unmarshal(envelope.Result, &list); err != nil {
		t.Fatal(err)
	}
	seen := map[uint64]bool{}
	for _, item := range list.List {
		seen[item.Id] = true
		if item.Images == nil {
			t.Fatalf("topic %d missing images field", item.Id)
		}
		switch item.Id {
		case 6104:
			if !reflect.DeepEqual(*item.Images, galleryFixture) {
				t.Fatalf("images = %#v", *item.Images)
			}
		case 6105:
			if len(*item.Images) != 0 {
				t.Fatalf("plain topic images = %#v, want []", *item.Images)
			}
		}
	}
	if !seen[6104] || !seen[6105] {
		t.Fatalf("list = %#v", list.List)
	}
}
