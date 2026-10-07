package forum

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultFeedCapabilityMatrixPreservesExplicitLatest(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(feed.Models()...); err != nil {
		t.Fatal(err)
	}
	preferences.Set("ranking.enabled", true)
	preferences.Set("feed.metrics.enabled", true)
	preferences.Set("feed.for_you.enabled", true)
	preferences.Set("feed.experiments.period", "route-matrix")
	preferences.Set("feed.experiments.salt", "route-matrix")
	feedconfig.SetRankReady(true)
	t.Cleanup(func() {
		preferences.Set("ranking.enabled", false)
		preferences.Set("feed.metrics.enabled", false)
		preferences.Set("feed.for_you.enabled", false)
		preferences.Set("feed.experiments.period", "default-entry-v1")
		preferences.Set("feed.experiments.salt", "default-entry-v1")
		feedconfig.SetRankReady(false)
	})
	uid := uint64(977000)
	for {
		hash := sha256.Sum256([]byte(fmt.Sprintf("route-matrix:entry:%d", uid)))
		if binary.BigEndian.Uint64(hash[:8])%100 < 20 {
			break
		}
		uid++
	}
	cases := []struct {
		name, cap, sort, want string
		json, guest, prefetch bool
	}{
		{name: "HTML default", want: "for_you"},
		{name: "v2 default", cap: "2", json: true, want: "for_you"},
		{name: "old root", json: true, want: ""},
		{name: "invalid capability", cap: "02", json: true, want: ""},
		{name: "old explicit recommendation", sort: "for_you", json: true, want: "latest"},
		{name: "v2 explicit latest", sort: "latest", cap: "2", json: true, want: "latest"},
		{name: "HTML explicit latest", sort: "latest", want: "latest"},
		{name: "guest root", cap: "2", json: true, guest: true, want: ""},
		{name: "prefetch root", cap: "2", json: true, prefetch: true, want: ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if !test.guest {
				c.Set("userId", uid)
			}
			if test.json {
				c.Request.Header.Set("X-Goose-Page", "true")
			}
			c.Request.Header.Set("X-Goose-Feed-Version", test.cap)
			if test.prefetch {
				c.Request.Header.Set("X-Goose-Prefetch", "1")
			}
			if got := personalizeDefault(c, test.sort); got != test.want {
				t.Fatalf("sort=%q want %q", got, test.want)
			}
		})
	}
}

func TestRecommendationFallbackDoesNotInjectAuthorsPrivatePendingTopics(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	preferences.Set("feed.for_you.enabled", false)
	preferences.Set("feed.metrics.enabled", false)
	const uid, topicID uint64 = 978901, 978902
	topic := topics.Entity{Id: topicID, UserId: uid, Status: 1, ProcessStatus: topics.ProcessStatusPending, FirstPostId: topicID, Title: "private pending"}
	post := posts.Entity{Id: topicID, TopicId: topicID, UserId: uid, PostNo: 1, ProcessStatus: posts.ProcessStatusPending}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&topic); conn.Unscoped().Delete(&post) })
	router := gin.New()
	router.GET("/", func(c *gin.Context) { c.Set("userId", uid); Home(c) })
	read := func(sort string) HomeProps {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/?sort="+sort, nil)
		req.Header.Set("X-Goose-Page", "true")
		req.Header.Set("X-Goose-Feed-Version", "2")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var payload struct {
			Props HomeProps `json:"props"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Props
	}
	recommended := read("for_you")
	for _, row := range recommended.Topics {
		if row.ID == topicID {
			t.Fatal("private author topic bypassed recommendation hard filters during fallback")
		}
	}
	latest := read("latest")
	for _, row := range latest.Topics {
		if row.ID == topicID {
			return
		}
	}
	t.Fatal("Latest lost the author's existing pending-publication feedback")
}
