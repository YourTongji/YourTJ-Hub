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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func seedHomeFeedAuthor(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	author := users.EntityComplete{Id: id, Username: fmt.Sprintf("u%d", id)}
	if err := conn.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&users.EntityComplete{}, id) })
}

func seedHomeFeedTopic(t *testing.T, conn *gorm.DB, id, author uint64, at time.Time) {
	t.Helper()
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

func homeFeedRequest(t *testing.T, loginUser uint64) []uint64 {
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
		seedHomeFeedAuthor(t, conn, id)
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
		seedHomeFeedTopic(t, conn, uint64(977010+i), author, at)
	}
	hotdataserve.ClearTopicListCache()
	t.Cleanup(hotdataserve.ClearTopicListCache)
	users.InvalidateBlockedPeersCache(viewer, 977099)
	t.Cleanup(func() { users.InvalidateBlockedPeersCache(viewer, 977099) })

	if got := homeFeedRequest(t, viewer); len(got) != 1 || got[0] != 977012 {
		t.Fatalf("blocked viewer topics=%v, want only the neutral topic 977012", got)
	}
	if got := homeFeedRequest(t, 0); len(got) != 3 {
		t.Fatalf("guest topics=%v, want all topics", got)
	}
	if got := homeFeedRequest(t, 977099); len(got) != 3 {
		t.Fatalf("unblocked viewer topics=%v, want all topics", got)
	}
}

// TestHomePayloadAppliesBlockChangesImmediately 覆盖 issue #1129 的语义护栏：
// 读者级屏蔽存在性缓存引入后，经生产写路径（userservice.SetBlockedUser）
// 新增/解除的屏蔽仍必须在下一次首页请求立即生效，不得等待缓存 TTL。
func TestHomePayloadAppliesBlockChangesImmediately(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &users.BlockEntity{}); err != nil {
		t.Fatal(err)
	}
	const viewer, blocked, neutral uint64 = 977101, 977102, 977103
	for _, id := range []uint64{viewer, blocked, neutral} {
		seedHomeFeedAuthor(t, conn, id)
	}
	at := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	seedHomeFeedTopic(t, conn, 977110, blocked, at.Add(time.Hour))
	seedHomeFeedTopic(t, conn, 977111, neutral, at)
	users.InvalidateBlockedPeersCache(viewer, blocked, neutral)
	t.Cleanup(func() { users.InvalidateBlockedPeersCache(viewer, blocked, neutral) })
	hotdataserve.ClearTopicListCache()
	t.Cleanup(hotdataserve.ClearTopicListCache)

	// 预热读者级缓存：无屏蔽关系的判定被缓存为 false。
	if got := homeFeedRequest(t, viewer); len(got) != 2 {
		t.Fatalf("pre-block topics=%v, want both topics", got)
	}

	// 新增屏蔽：下一次请求必须立即过滤，不能等待缓存 TTL。
	if err := userservice.SetBlockedUser(viewer, blocked, true); err != nil {
		t.Fatalf("block author: %v", err)
	}
	t.Cleanup(func() { _ = userservice.SetBlockedUser(viewer, blocked, false) })
	if got := homeFeedRequest(t, viewer); len(got) != 1 || got[0] != 977111 {
		t.Fatalf("topics after new block=%v, want only the neutral topic", got)
	}

	// 解除屏蔽：同样立即恢复。
	if err := userservice.SetBlockedUser(viewer, blocked, false); err != nil {
		t.Fatalf("unblock author: %v", err)
	}
	if got := homeFeedRequest(t, viewer); len(got) != 2 {
		t.Fatalf("topics after unblock=%v, want both topics", got)
	}

	// 反向：作者拉黑读者同样在下一次请求立即生效。
	if err := userservice.SetBlockedUser(blocked, viewer, true); err != nil {
		t.Fatalf("author blocks viewer: %v", err)
	}
	t.Cleanup(func() { _ = userservice.SetBlockedUser(blocked, viewer, false) })
	if got := homeFeedRequest(t, viewer); len(got) != 1 || got[0] != 977111 {
		t.Fatalf("topics after reverse block=%v, want only the neutral topic", got)
	}
}
