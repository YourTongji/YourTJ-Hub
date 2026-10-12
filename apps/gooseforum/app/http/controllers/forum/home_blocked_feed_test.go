package forum

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
)

// TestHomePayloadHidesBlockedAuthorsInEitherDirection 覆盖 HTTP 层验收：
// 首页对登录读者应用双向屏蔽过滤（含反向屏蔽），游客与无屏蔽读者不受影响。
func TestHomePayloadHidesBlockedAuthorsInEitherDirection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &users.BlockEntity{}); err != nil {
		t.Fatal(err)
	}
	const viewer, blocked, blocker, neutral uint64 = 977001, 977002, 977003, 977004
	for _, id := range []uint64{blocked, blocker, neutral} {
		author := users.EntityComplete{Id: id, Username: fmt.Sprintf("u%d", id)}
		if err := conn.Create(&author).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(&users.EntityComplete{}, id) })
	}
	for _, block := range []users.BlockEntity{
		{OwnerID: viewer, TargetUserID: blocked},
		{OwnerID: blocker, TargetUserID: viewer},
	} {
		row := block
		if err := conn.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			conn.Delete(&users.BlockEntity{}, "owner_id = ? AND target_user_id = ?", row.OwnerID, row.TargetUserID)
		})
	}
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	for i, author := range []uint64{blocked, blocker, neutral} {
		id := uint64(977010 + i)
		topic := topics.Entity{Id: id, UserId: author, Title: fmt.Sprintf("topic %d", id), Status: 1, TopicType: topics.TopicTypeForum, FirstPostId: id, CreatedAt: at, UpdatedAt: at}
		post := posts.Entity{Id: id, TopicId: id, PostNo: 1, UserId: author, Content: "body", CreatedAt: at, UpdatedAt: at}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatal(err)
		}
		if err := conn.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			conn.Unscoped().Delete(&posts.Entity{}, id)
			conn.Unscoped().Delete(&topics.Entity{}, id)
		})
	}
	hotdataserve.ClearTopicListCache()
	t.Cleanup(hotdataserve.ClearTopicListCache)

	request := func(loginUser uint64) []uint64 {
		t.Helper()
		router := gin.New()
		router.GET("/", func(c *gin.Context) {
			if loginUser != 0 {
				c.Set("userId", loginUser)
			}
			Home(c)
		})
		req := httptest.NewRequest(http.MethodGet, "/?sort=latest", nil)
		req.Header.Set("X-Goose-Page", "true")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Props HomeProps `json:"props"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		ids := make([]uint64, 0, len(payload.Props.Topics))
		for _, item := range payload.Props.Topics {
			ids = append(ids, item.ID)
		}
		return ids
	}

	if got := request(viewer); len(got) != 1 || got[0] != 977012 {
		t.Fatalf("blocked viewer topics=%v, want only the neutral topic 977012", got)
	}
	if got := request(0); len(got) != 3 {
		t.Fatalf("guest topics=%v, want all topics", got)
	}
	if got := request(977099); len(got) != 3 {
		t.Fatalf("unblocked viewer topics=%v, want all topics", got)
	}
}
