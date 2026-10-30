package forum

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/gin-gonic/gin"
)

func TestTopicDetailGalleryFollowsFirstPostVisibility(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/p/post/:id", TopicDetail)
	for index, scenario := range []struct {
		name        string
		visibility  string
		process     int8
		removeFirst bool
		visible     bool
	}{
		{name: "public", visibility: posts.VisibilityActive, visible: true},
		{name: "author deleted", visibility: posts.VisibilityUserDeleted},
		{name: "moderator removed", visibility: posts.VisibilityModeratorRemoved},
		{name: "hidden", visibility: posts.VisibilityActive, process: posts.ProcessStatusBlocked},
		{name: "missing first post", visibility: posts.VisibilityActive, removeFirst: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			id := uint64(9812310 + index*10)
			image := fmt.Sprintf("/file/img/gallery-visibility-%d.png", id)
			topic := topics.Entity{Id: id, UserId: id, Title: "gallery", FirstPostId: id, PostSeq: 2, ReplyCount: 1, Status: 1, FirstImageURL: image, ImageUrls: []string{image}}
			first := posts.Entity{Id: id, TopicId: id, PostNo: 1, UserId: id, Content: "first", VisibilityStatus: scenario.visibility, ProcessStatus: scenario.process}
			reply := posts.Entity{Id: id + 1, TopicId: id, PostNo: 2, UserId: id + 1, Content: "public reply"}
			for _, row := range []any{&topic, &first, &reply} {
				if err := db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { db.Unscoped().Delete(row) })
			}
			if scenario.removeFirst {
				if err := db.Unscoped().Delete(&first).Error; err != nil {
					t.Fatal(err)
				}
			}
			for _, json := range []bool{true, false} {
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/p/post/%d", id), nil)
				if json {
					req.Header.Set("X-Goose-Page", "true")
				}
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Fatalf("status %d: %s", w.Code, w.Body.String())
				}
				if got := strings.Contains(w.Body.String(), image); got != scenario.visible {
					t.Fatalf("json=%t gallery present=%t, want %t", json, got, scenario.visible)
				}
			}
			stored := topics.GetSimple(id)
			if stored.FirstImageURL != image || len(stored.ImageUrls) != 1 {
				t.Fatal("read changed stored gallery")
			}
		})
	}
}
