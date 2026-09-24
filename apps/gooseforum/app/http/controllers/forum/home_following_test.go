package forum

import (
	"context"
	"encoding/json"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/gin-gonic/gin"
)

func TestFollowingHomeGuestRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", Home)
	for _, pageRequest := range []bool{false, true} {
		req := httptest.NewRequest(http.MethodGet, "/?sort=following", nil)
		if pageRequest {
			req.Header.Set("X-Goose-Page", "true")
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if pageRequest {
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("guest following status = %d, want 401", recorder.Code)
			}
			var result component.ResultStruct
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.MessageCode != component.MessageAuthRequired {
				t.Fatalf("guest following message = %q", result.MessageCode)
			}
		} else if recorder.Code != http.StatusFound || !strings.HasPrefix(recorder.Header().Get("Location"), "/login?redirect=") {
			t.Fatalf("guest HTML must redirect to login: %d %q", recorder.Code, recorder.Header().Get("Location"))
		}
		if !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("personalized following response must not be cached")
		}
	}
}

func TestFollowingHomePayloadCursorAndLocale(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &userFollow.Entity{}); err != nil {
		t.Fatal(err)
	}
	const viewer, author uint64 = 976000, 976001
	edge := userFollow.Entity{UserId: viewer, FollowUserId: author, Status: 1}
	if err := db.Create(&edge).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Delete(&edge)
		db.Unscoped().Where("user_id = ?", author).Delete(&posts.Entity{})
		db.Unscoped().Where("user_id = ?", author).Delete(&topics.Entity{})
	})
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	for id := uint64(976100); id < 976121; id++ {
		if err := db.Create(&topics.Entity{Id: id, UserId: author, Title: "followed", Status: 1, FirstPostId: id, CreatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&posts.Entity{Id: id, TopicId: id, PostNo: 1, UserId: author, Content: "body"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	router := gin.New()
	router.GET("/", func(c *gin.Context) { c.Set("userId", viewer); Home(c) })
	request := func(path string) struct {
		Component string    `json:"component"`
		Props     HomeProps `json:"props"`
		Meta      PageMeta  `json:"meta"`
	} {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Goose-Page", "true")
		req.Header.Set("Accept-Language", "en")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != 200 {
			t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Component string    `json:"component"`
			Props     HomeProps `json:"props"`
			Meta      PageMeta  `json:"meta"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("personalized payload is cacheable")
		}
		return payload
	}
	first := request("/?sort=following")
	if first.Component != string(PageComponentHome) || first.Props.Sort != "following" || len(first.Props.Topics) != 20 {
		t.Fatalf("first payload: %+v", first)
	}
	followingTab := false
	for _, tab := range first.Props.Tabs {
		if tab.Key == "following" {
			followingTab = tab.Active && tab.Label == "Following" && tab.URL == "/?sort=following"
		}
	}
	if !followingTab {
		t.Fatalf("localized active following tab missing: %+v", first.Props.Tabs)
	}
	next, err := url.Parse(first.Props.Pagination.NextURL)
	if err != nil || next.Query().Get("cursor") == "" || next.Query().Get("sort") != "following" || next.Query().Get("page") != "2" {
		t.Fatalf("next URL: %v %v", next, err)
	}
	if first.Meta.Robots != "noindex, nofollow" || first.Meta.NextURL != next.String() || first.Meta.PrevURL != "" {
		t.Fatalf("private feed metadata: %+v", first.Meta)
	}
	second := request(next.String())
	if len(second.Props.Topics) != 1 || second.Props.Topics[0].ID != 976100 || second.Props.Pagination.HasNext || second.Props.Pagination.NextURL != "" {
		t.Fatalf("second payload: %+v", second)
	}
}

func TestFollowingHomeRejectsInvalidCursorAndReadFailure(t *testing.T) {
	router := gin.New()
	router.GET("/", func(c *gin.Context) { c.Set("userId", uint64(976000)); Home(c) })
	for _, path := range []string{"/?sort=following&cursor=garbage", "/?sort=following&page=2"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Goose-Page", "true")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("invalid cursor returned %d: %s", recorder.Code, recorder.Body.String())
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/?sort=following", nil).WithContext(ctx)
	req.Header.Set("X-Goose-Page", "true")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("read failure returned %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestHomeGuestFollowingTabUsesLoginURL(t *testing.T) {
	tabs := buildHomeTabs("latest", 0, "en")
	for _, tab := range tabs {
		if tab.Key == "following" {
			if tab.Label != "Following" || tab.URL != followingLoginURL() {
				t.Fatalf("guest tab: %+v", tab)
			}
			return
		}
	}
	t.Fatal("following entry missing")
}
