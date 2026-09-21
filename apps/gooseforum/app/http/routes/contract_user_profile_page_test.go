package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupProfilePageContractTest 在共享 harness 之上补齐 /u/:userId 视图路由
// （与 route4api.go viewRoute 注册一致：JWTAuth 可选登录 + X-Goose-Page JSON）。
func setupProfilePageContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&userFollow.Entity{}); err != nil {
		t.Fatalf("migrate userFollow contract table: %v", err)
	}
	view := router.Group("")
	view.Use(middleware.JWTAuth)
	view.GET("/u/:userId", forum.UserProfile)
	view.GET("/u/:userId/:section", forum.UserProfile)
	view.GET("/u/:userId/:section/:subsection", forum.UserProfile)
	return conn, router
}

// TestUserProfilePagePayloadNonNullLists 锁定 /u/:id 页面载荷的数组非空契约。
//
// 回归背景（2026-09-06 生产实捕）：summary 区块下 activityTabs 与 user.badges
// 曾以 Go nil 切片序列化为 JSON null（违反 payload.ts 的非空数组声明），
// 移动端非空镜像解析抛错被吞 → Profile 页显示「Failed to parse page data」；
// 无徽章用户的 props.badges 同样存在 nil 风险。本测试用无徽章的全新用户
// 覆盖全部三处 null 源。
func TestUserProfilePagePayloadNonNullLists(t *testing.T) {
	conn, router := setupProfilePageContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())

	req := httptest.NewRequest(http.MethodGet, "/u/"+strconv.FormatUint(alice.Id, 10), nil)
	req.Header.Set("X-Goose-Page", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var payload struct {
		Component string                     `json:"component"`
		Props     map[string]json.RawMessage `json:"props"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode profile SSR payload: %v", err)
	}
	if payload.Component != "user.profile" {
		t.Fatalf("component = %q, want user.profile", payload.Component)
	}
	// UserProfileProps 的全部数组字段按契约（payload.ts）都是非空数组；
	// 任一字段序列化为 null 都会让移动端非空镜像解析失败。
	listFields := []string{
		"tabs", "activityTabs", "badges", "topics", "activities",
		"likes", "bookmarks", "following", "followers",
	}
	for _, field := range listFields {
		raw, ok := payload.Props[field]
		if !ok {
			t.Fatalf("props.%s missing from payload", field)
		}
		if string(raw) == "null" {
			t.Fatalf("props.%s serialized as null, want [] (non-empty-array contract)", field)
		}
	}
	var user struct {
		Badges json.RawMessage `json:"badges"`
	}
	if err := json.Unmarshal(payload.Props["user"], &user); err != nil {
		t.Fatalf("decode props.user: %v", err)
	}
	if string(user.Badges) == "null" {
		t.Fatalf("props.user.badges serialized as null, want [] (non-empty-array contract)")
	}
}

func TestUserProfileConnectionSectionsUseCanonicalPagedProps(t *testing.T) {
	conn, router := setupProfilePageContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	var firstPeerID uint64
	for i := uint64(1); i <= 25; i++ {
		peer := createHTTPContractUser(t, conn, contractTestID()+i)
		if i == 1 {
			firstPeerID = peer.Id
		}
		if err := conn.Create(&userFollow.Entity{UserId: alice.Id, FollowUserId: peer.Id, Status: 1}).Error; err != nil {
			t.Fatalf("create following relation: %v", err)
		}
	}

	requestPayload := func(path string) map[string]json.RawMessage {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Goose-Page", "true")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var payload struct {
			Props map[string]json.RawMessage `json:"props"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode %s payload: %v", path, err)
		}
		return payload.Props
	}

	props := requestPayload("/u/" + strconv.FormatUint(alice.Id, 10) + "/following")
	var section, activityTab string
	if err := json.Unmarshal(props["section"], &section); err != nil {
		t.Fatalf("decode canonical section: %v", err)
	}
	if err := json.Unmarshal(props["activityTab"], &activityTab); err != nil {
		t.Fatalf("decode canonical activityTab: %v", err)
	}
	if section != "following" || activityTab != "following" {
		t.Fatalf("canonical props section/activityTab = %q/%q", section, activityTab)
	}
	var following []json.RawMessage
	if err := json.Unmarshal(props["following"], &following); err != nil {
		t.Fatalf("decode following list: %v", err)
	}
	if len(following) != 24 {
		t.Fatalf("following page length = %d, want 24", len(following))
	}
	var pagination struct {
		Page     int    `json:"page"`
		NextPage int    `json:"nextPage"`
		HasNext  bool   `json:"hasNext"`
		NextURL  string `json:"nextUrl"`
	}
	if err := json.Unmarshal(props["pagination"], &pagination); err != nil {
		t.Fatalf("decode following pagination: %v", err)
	}
	if pagination.Page != 1 || pagination.NextPage != 2 || !pagination.HasNext || pagination.NextURL != "/u/"+strconv.FormatUint(alice.Id, 10)+"/following?page=2" {
		t.Fatalf("following pagination = %+v", pagination)
	}

	legacyProps := requestPayload("/u/" + strconv.FormatUint(alice.Id, 10) + "/activity/following")
	if err := json.Unmarshal(legacyProps["section"], &section); err != nil {
		t.Fatalf("decode legacy section: %v", err)
	}
	if section != "following" {
		t.Fatalf("legacy section = %q, want following", section)
	}

	pageTwo := requestPayload("/u/" + strconv.FormatUint(alice.Id, 10) + "/following?page=2")
	if err := json.Unmarshal(pageTwo["pagination"], &pagination); err != nil {
		t.Fatalf("decode page two pagination: %v", err)
	}
	if pagination.Page != 2 || pagination.NextPage != 0 || pagination.HasNext || pagination.NextURL != "" {
		t.Fatalf("following page two pagination = %+v", pagination)
	}

	if err := conn.Create(&userFollow.Entity{UserId: firstPeerID, FollowUserId: alice.Id, Status: 1}).Error; err != nil {
		t.Fatalf("create follower relation: %v", err)
	}
	followerProps := requestPayload("/u/" + strconv.FormatUint(alice.Id, 10) + "/followers")
	if err := json.Unmarshal(followerProps["section"], &section); err != nil {
		t.Fatalf("decode followers section: %v", err)
	}
	if section != "followers" {
		t.Fatalf("followers section = %q, want followers", section)
	}
	var followers []json.RawMessage
	if err := json.Unmarshal(followerProps["followers"], &followers); err != nil {
		t.Fatalf("decode followers list: %v", err)
	}
	if len(followers) != 1 {
		t.Fatalf("followers page length = %d, want 1", len(followers))
	}
}
