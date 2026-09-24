package forum

import (
	"encoding/json"
	"testing"
	"time"

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
	topic := topics.Entity{Id: 978501, UserId: author.Id, Title: "Preview topic", Excerpt: "Topic summary", FirstImageURL: "/file/img/preview.png", Status: 1}
	hidden := topics.Entity{Id: 978502, UserId: author.Id, Title: "Hidden", Status: 1, ProcessStatus: 1}
	reply := posts.Entity{Id: 978501, TopicId: topic.Id, PostNo: 7, UserId: author.Id, Content: "**Readable** reply"}
	anonymous := posts.Entity{Id: 978502, TopicId: topic.Id, PostNo: 8, UserId: author.Id, Content: "Anonymous reply", IsAnonymous: true}
	hiddenReply := posts.Entity{Id: 978503, TopicId: hidden.Id, PostNo: 1, UserId: author.Id, Content: "Hidden topic body"}
	blockedReply := posts.Entity{Id: 978504, TopicId: topic.Id, PostNo: 9, UserId: author.Id, Content: "Blocked body", ProcessStatus: 1}
	for _, row := range []any{&author, &topic, &hidden, &reply, &anonymous, &hiddenReply, &blockedReply} {
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
	})
	if len(notifications) != 4 || notifications[0].Actor.AvatarURL != author.GetWebAvatarUrl() || notifications[0].Content != "Readable reply" {
		t.Fatalf("notification preview: %+v", notifications)
	}
	if notifications[1].Content != "" || notifications[2].Content != "" || notifications[3].Content != "" {
		t.Fatal("inconsistent hidden notification target exposed content")
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
