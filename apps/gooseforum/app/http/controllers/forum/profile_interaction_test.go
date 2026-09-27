package forum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
)

func TestProfileViewerInteractionState(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}, &topicUserAction.Entity{}, &postUserAction.Entity{}, &userActivities.Entity{}); err != nil {
		t.Fatal(err)
	}
	const owner, viewer, topicID, postID = uint64(979201), uint64(979202), uint64(979203), uint64(979204)
	now := time.Now()
	user := users.EntityComplete{Id: owner, Username: "profile-state-owner"}
	rows := []any{&user, &topics.Entity{Id: topicID, UserId: owner, Status: 1, FirstPostId: topicID, CreatedAt: now}, &posts.Entity{Id: topicID, TopicId: topicID, PostNo: 1, UserId: owner}, &posts.Entity{Id: postID, TopicId: topicID, PostNo: 2, UserId: owner}, &userActivities.Entity{Id: topicID, UserId: owner, Action: 2, SubjectType: userActivities.SubjectTopic, SubjectId: topicID, CreatedAt: now}, &userActivities.Entity{Id: postID, UserId: owner, Action: 5, SubjectType: userActivities.SubjectPost, SubjectId: postID, CreatedAt: now}}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Unscoped().Delete(row) })
	}
	t.Cleanup(func() {
		db.Where("user_id IN ?", []uint64{owner, viewer}).Delete(&topicUserAction.Entity{})
		db.Where("user_id IN ?", []uint64{owner, viewer}).Delete(&postUserAction.Entity{})
	})
	topicUserAction.SetLiked(viewer, topicID, true)
	topicUserAction.SetBookmarked(owner, topicID, true)
	postUserAction.SetBookmarked(viewer, postID, true)
	for _, viewerID := range []uint64{viewer, 0} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/u/979201/activity", nil)
		c.Set("userId", viewerID)
		for _, tab := range []string{userProfileActivityTopics, userProfileActivityTimeline} {
			props := buildUserProfileProps(c, user, userProfileSectionActivity, tab)
			raw, err := json.Marshal(props)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			key := "topics"
			if tab == userProfileActivityTimeline {
				key = "activities"
			}
			items := got[key].([]any)
			if len(items) == 0 {
				t.Fatalf("empty %s", key)
			}
			for _, item := range items {
				row := item.(map[string]any)
				if viewerID == 0 {
					if _, ok := row["liked"]; ok {
						t.Fatalf("guest state: %s", raw)
					}
					continue
				}
				reply := row["subjectType"] == userActivities.SubjectPost
				if row["liked"] != !reply || row["bookmarked"] != reply {
					t.Fatalf("%s viewer state: %s", key, raw)
				}
			}
		}
	}
	// Personal state must not revive actions on hidden or deleted content.
	if err := db.Model(&topics.Entity{}).Where("id = ?", topicID).Update("process_status", 1).Error; err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/u/979201/activity", nil)
	c.Set("userId", viewer)
	hidden := buildUserProfileProps(c, user, userProfileSectionActivity, userProfileActivityTimeline)
	for _, row := range hidden.Activities {
		if row.Liked != nil || row.Bookmarked != nil {
			t.Fatal("hidden parent exposed interaction controls")
		}
	}

}
