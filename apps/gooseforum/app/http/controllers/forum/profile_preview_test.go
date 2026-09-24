package forum

import (
	"encoding/json"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestProfilePreviewsUseContentAuthorAndRespectVisibility(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	author := users.EntityComplete{Id: 978501, Username: "preview-author", Nickname: "Preview Author", AvatarUrl: "/avatar/preview.png"}
	topic := topics.Entity{Id: 978501, UserId: author.Id, Title: "Preview topic", Excerpt: "Topic summary", FirstImageURL: "/file/img/preview.png", Status: 1, FirstPostId: 978505}
	hidden := topics.Entity{Id: 978502, UserId: author.Id, Title: "Hidden", Status: 1, ProcessStatus: 1}
	reply := posts.Entity{Id: 978501, TopicId: topic.Id, PostNo: 7, UserId: author.Id, Content: "**Readable** reply"}
	anonymous := posts.Entity{Id: 978502, TopicId: topic.Id, PostNo: 8, UserId: author.Id, Content: "Anonymous reply", IsAnonymous: true}
	hiddenReply := posts.Entity{Id: 978503, TopicId: hidden.Id, PostNo: 1, UserId: author.Id, Content: "Hidden topic body"}
	blockedReply := posts.Entity{Id: 978504, TopicId: topic.Id, PostNo: 9, UserId: author.Id, Content: "Blocked body", ProcessStatus: 1}
	first := posts.Entity{Id: 978505, TopicId: topic.Id, PostNo: 1, UserId: author.Id, Content: "Topic body"}
	for _, row := range []any{&first, &author, &topic, &hidden, &reply, &anonymous, &hiddenReply, &blockedReply} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Unscoped().Delete(row) })
	}
	notifications := BuildNotificationPayloads([]*eventNotification.Entity{nil, {
		EventType: eventNotification.EventTypeLike,
		Payload:   eventNotification.NotificationPayload{ActorId: author.Id, ActorName: author.Username, TopicId: topic.Id, PostId: reply.Id, PostNo: reply.PostNo},
	}, {EventType: eventNotification.EventTypeLike, Payload: eventNotification.NotificationPayload{ActorId: author.Id, TopicId: hidden.Id, PostId: hiddenReply.Id}},
		{EventType: eventNotification.EventTypeLike, Payload: eventNotification.NotificationPayload{ActorId: author.Id, TopicId: topic.Id, PostId: blockedReply.Id}},
		{EventType: eventNotification.EventTypeLike, Payload: eventNotification.NotificationPayload{ActorId: author.Id, TopicId: hidden.Id, PostId: reply.Id}},
		{EventType: eventNotification.EventTypeLike, Payload: eventNotification.NotificationPayload{ActorId: author.Id, TopicId: topic.Id, PostId: reply.Id, TemplateParams: eventNotification.NotificationTemplateParams{Preview: "Stored preview"}}},
	})
	if len(notifications) != 5 || notifications[0].Actor.AvatarURL != author.GetWebAvatarUrl() || notifications[0].Content != "Readable reply" {
		t.Fatalf("notification preview: %+v", notifications)
	}
	if notifications[1].Content != "" || notifications[2].Content != "" || notifications[3].Content != "" {
		t.Fatal("inconsistent hidden notification target exposed content")
	}
	if notifications[4].Content != "" || notifications[4].Payload.TemplateParams.Preview != "Stored preview" {
		t.Fatal("stored preview was replaced")
	}
	likes := buildUserLikes([]topicUserAction.LikedTopicRef{{ID: 1, TopicID: topic.Id, LikedAt: time.Now()}, {ID: 2, TopicID: hidden.Id}})
	raw, err := json.Marshal(likes)
	if err != nil {
		t.Fatal(err)
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["excerpt"] != "Topic summary" || list[0]["thumbnailUrl"] != "/file/img/preview.png" {
		t.Fatalf("like preview: %s", raw)
	}
	a, ok := list[0]["author"].(map[string]any)
	if !ok || a["id"] != float64(author.Id) || a["avatarUrl"] != author.GetWebAvatarUrl() {
		t.Fatalf("like author: %s", raw)
	}
	bookmarks := buildBookmarkPayloads([]mergedBookmarkRef{{kind: "topic", topicID: topic.Id}, {kind: "post", postID: reply.Id}, {kind: "post", postID: anonymous.Id}, {kind: "topic", topicID: hidden.Id}})
	raw, err = json.Marshal(bookmarks)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[1]["excerpt"] != "Readable reply" || list[1]["url"] != "/p/post/978501/7#post-978501" {
		t.Fatalf("bookmark preview: %s", raw)
	}
	if _, ok := list[1]["author"].(map[string]any); !ok {
		t.Fatalf("reply author missing: %s", raw)
	}
	if list[2]["author"] != nil {
		t.Fatalf("anonymous identity exposed: %s", raw)
	}
}

// A retained tombstone still has a normal process status and a NULL deleted_at.
// Preview reads must honor lifecycle visibility as well as the public first post.
func TestProfilePreviewsExcludeTombstonesAndHiddenFirstPosts(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"reply tombstone", "topic tombstone", "blocked first post", "deleted first post", "tombstone first post", "missing first post", "foreign first post"} {
		t.Run(scenario, func(t *testing.T) {
			topic := topics.Entity{Id: 978510, Status: 1, FirstPostId: 978510, Title: "Private title", Excerpt: "Private summary", FirstImageURL: "/private.png"}
			first := posts.Entity{Id: 978510, TopicId: topic.Id, PostNo: 1, Content: "Private first body"}
			reply := posts.Entity{Id: 978511, TopicId: topic.Id, PostNo: 2, Content: "Private reply body"}
			switch scenario {
			case "reply tombstone":
				reply.VisibilityStatus = posts.VisibilityUserDeleted
				reply.RetentionStatus = posts.RetentionRecoverable
			case "topic tombstone":
				topic.VisibilityStatus = topics.VisibilityUserDeleted
			case "blocked first post":
				first.ProcessStatus = posts.ProcessStatusBlocked
			case "deleted first post":
				first.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
			case "tombstone first post":
				first.VisibilityStatus = posts.VisibilityUserDeleted
			case "missing first post":
				topic.FirstPostId = 978599
			case "foreign first post":
				first.TopicId = 978599
			}
			for _, row := range []any{&topic, &first, &reply} {
				if err := db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { db.Unscoped().Delete(row) })
			}
			notifications := BuildNotificationPayloads([]*eventNotification.Entity{{EventType: eventNotification.EventTypeLike, Payload: eventNotification.NotificationPayload{TopicId: topic.Id, PostId: reply.Id}}})
			if len(notifications) != 1 || notifications[0].Content != "" {
				t.Errorf("non-public reply rehydrated: %+v", notifications)
			}
			if got := buildBookmarkPayloads([]mergedBookmarkRef{{kind: "post", postID: reply.Id}}); len(got) != 0 {
				t.Errorf("non-public reply bookmark: %+v", got)
			}
			if scenario != "reply tombstone" {
				if got := buildUserLikes([]topicUserAction.LikedTopicRef{{TopicID: topic.Id}}); len(got) != 0 {
					t.Errorf("non-public liked topic: %+v", got)
				}
				if got := buildUserBookmarks([]topicUserAction.BookmarkedTopicRef{{TopicID: topic.Id}}); len(got) != 0 {
					t.Errorf("non-public legacy bookmark: %+v", got)
				}
				if got := buildBookmarkPayloads([]mergedBookmarkRef{{kind: "topic", topicID: topic.Id}}); len(got) != 0 {
					t.Errorf("non-public topic bookmark: %+v", got)
				}
			}
		})
	}
}
