package pushpolicy

import (
	"encoding/json"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"testing"
	"time"
)

func setup(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &taskQueue.Entity{}, &eventNotification.Entity{}, &messages.Entity{}, &imUserChatConfigs.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{98001, 98002} {
		conn.Unscoped().Where("id = ?", id).Delete(&users.EntityComplete{})
		if err := conn.Create(&users.EntityComplete{Id: id, Username: "policy-user-" + string(rune(id)), Email: ""}).Error; err != nil {
			t.Fatal(err)
		}
	}
}
func TestPartialPreferencesPreserveSelectiveSystemDefault(t *testing.T) {
	setup(t)
	off := false
	if _, err := Update(98001, Changes{Chat: &off}); err != nil {
		t.Fatal(err)
	}
	for typ, want := range map[string]bool{"chat_message": false, "mention": true, "like": false, "follow": false, "review_pending": false, "review_approved": true, "review_rejected": true, "badge": false, "system": false} {
		got, err := Allowed(98001, typ)
		if err != nil || got != want {
			t.Errorf("%s = %v,%v want %v", typ, got, err, want)
		}
	}
	on := true
	if _, err := Update(98001, Changes{System: &on}); err != nil {
		t.Fatal(err)
	}
	if got, _ := Allowed(98001, "unregistered_future"); got {
		t.Fatal("unknown event must remain silent even with system opt-in")
	}
	if got, _ := Allowed(98001, "badge"); !got {
		t.Fatal("explicit system opt-in lost")
	}
}
func TestChatSourceChecksAndRollback(t *testing.T) {
	setup(t)
	conn := db.Connect()
	conn.Where("id = ?", 98003).Delete(&messages.Entity{})
	conn.Where("conv_id = ?", 98004).Delete(&imUserChatConfigs.Entity{})
	if err := conn.Create(&messages.Entity{Id: 98003, ConvId: 98004, SenderId: 98002, Content: "private must not be loaded"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&imUserChatConfigs.Entity{UserId: 98001, PeerId: 98002, ConvId: 98004}).Error; err != nil {
		t.Fatal(err)
	}
	task := Task{UserId: 98001, SourceKind: ChatMessage, SourceId: 98003}
	_, chat, err := Resolve(task)
	if err != nil || chat == nil || chat.Content != "" {
		t.Fatalf("safe source load: %+v %v", chat, err)
	}
	for _, column := range []string{"is_muted", "is_deleted"} {
		conn.Model(&imUserChatConfigs.Entity{}).Where("user_id = ? AND conv_id = ?", 98001, 98004).Update(column, 1)
		_, chat, err = Resolve(task)
		if err != nil || chat != nil {
			t.Fatalf("%s did not suppress", column)
		}
		conn.Model(&imUserChatConfigs.Entity{}).Where("user_id = ? AND conv_id = ?", 98001, 98004).Update(column, 0)
	}
	conn.Model(&messages.Entity{}).Where("id = ?", 98003).Update("is_read", 1)
	_, chat, err = Resolve(task)
	if err != nil || chat != nil {
		t.Fatal("read source sent")
	}
	task.UserId = 98002
	_, chat, err = Resolve(task)
	if err != nil || chat != nil {
		t.Fatal("wrong owner sent")
	}
	conn.Where("dedup_key = ?", "testpush.:chat_message:98003:98001").Delete(&taskQueue.Entity{})
	rollback := conn.Transaction(func(tx *gorm.DB) error {
		if err := Enqueue(tx, "testpush.", 98001, 98003, ChatMessage); err != nil {
			return err
		}
		return gorm.ErrInvalidTransaction
	})
	if rollback == nil {
		t.Fatal("expected rollback")
	}
	var count int64
	conn.Model(&taskQueue.Entity{}).Where("dedup_key = ?", "testpush.:chat_message:98003:98001").Count(&count)
	if count != 0 {
		t.Fatal("rolled back chat queued")
	}
	for i := 0; i < 2; i++ {
		if err := Enqueue(conn, "testpush.", 98001, 98003, ChatMessage); err != nil {
			t.Fatal(err)
		}
	}
	conn.Model(&taskQueue.Entity{}).Where("dedup_key = ?", "testpush.:chat_message:98003:98001").Count(&count)
	if count != 1 {
		t.Fatal("duplicate outbox task")
	}
	var stored taskQueue.Entity
	conn.Where("dedup_key = ?", "testpush.:chat_message:98003:98001").First(&stored)
	if stored.Type != "testpush." {
		t.Fatalf("task type = %q, want the channel family constant", stored.Type)
	}
	var payload map[string]any
	if json.Unmarshal([]byte(stored.TaskJson), &payload) != nil || payload["sourceKind"] != ChatMessage {
		t.Fatal("wrong source envelope")
	}
}

func TestLowValueThrottlePreservesHighValue(t *testing.T) {
	setup(t)
	conn := db.Connect()
	conn.Where("type = ?", "throttle-test").Delete(&taskQueue.Entity{})
	now := time.Now()
	first := taskQueue.Entity{Type: "throttle-test", Status: taskQueue.StatusSuccess, ProcessedAt: now, ScheduleGroup: "web:98001:like:123"}
	if err := conn.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	second := taskQueue.Entity{Type: "throttle-test", Status: taskQueue.StatusRunning, ProcessedAt: now}
	if err := conn.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	payload := Task{UserId: 98001}
	n := eventNotification.Entity{EventType: "like", Payload: eventNotification.NotificationPayload{TopicId: 123}}
	blocked, err := Throttle(&second, payload, n, "web")
	if err != nil || !blocked {
		t.Fatalf("low value not throttled %v %v", blocked, err)
	}
	n.EventType = "mention"
	blocked, err = Throttle(&second, payload, n, "web")
	if err != nil || blocked {
		t.Fatal("mention throttled")
	}
	n.EventType = "like"
	n.Payload.TopicId = 124
	blocked, err = Throttle(&second, payload, n, "web")
	if err != nil || blocked {
		t.Fatal("unrelated target throttled")
	}
}

// 节流计数必须限定在被节流行自己的任务族内：未来的任务族若复用同一
// schedule_group 字符串，不得静默抑制推送。
func TestLowValueThrottleIgnoresOtherTaskFamilies(t *testing.T) {
	setup(t)
	conn := db.Connect()
	for _, typ := range []string{"agent-webhook.9", "webpush.98001"} {
		conn.Where("type = ?", typ).Delete(&taskQueue.Entity{})
	}
	now := time.Now()
	other := taskQueue.Entity{Type: "agent-webhook.9", Status: taskQueue.StatusSuccess, ProcessedAt: now, ScheduleGroup: "web:98001:like:123"}
	if err := conn.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	second := taskQueue.Entity{Type: "webpush.98001", Status: taskQueue.StatusRunning, ProcessedAt: now}
	if err := conn.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	payload := Task{UserId: 98001}
	n := eventNotification.Entity{EventType: "like", Payload: eventNotification.NotificationPayload{TopicId: 123}}
	blocked, err := Throttle(&second, payload, n, "web")
	if err != nil || blocked {
		t.Fatalf("another task family suppressed this push: blocked=%v err=%v", blocked, err)
	}
}

func TestCompleteEventPolicyGroups(t *testing.T) {
	for typ, want := range map[string]string{"mention": "mentionsAndReplies", "post_reply": "mentionsAndReplies", "comment": "mentionsAndReplies", "like": "activity", "follow": "following", "topic_post": "following", "wiki_updated": "following", "badge": "system", "system": "system", "review_pending": "system", "review_approved": "system", "review_rejected": "system", "chat_message": "chat", "inboxmail.delivery": "inboxmail", "unregistered_future": "unknown"} {
		if got := Group(typ); got != want {
			t.Fatalf("%s group %s want %s", typ, got, want)
		}
	}
}

func TestNotificationSourceRejectsDraftAndPurgedContent(t *testing.T) {
	setup(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	const topicID, postID, notificationID = uint64(98901), uint64(98902), uint64(98903)
	conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
	conn.Unscoped().Where("id = ?", postID).Delete(&posts.Entity{})
	conn.Where("id = ?", notificationID).Delete(&eventNotification.Entity{})
	if err := conn.Create(&topics.Entity{Id: topicID, Title: "private draft", FirstPostId: postID, Status: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&eventNotification.Entity{Id: notificationID, UserId: 98001, EventType: eventNotification.EventTypeMention, Payload: eventNotification.NotificationPayload{TopicId: topicID, PostId: postID}}).Error; err != nil {
		t.Fatal(err)
	}
	task := Task{UserId: 98001, NotificationId: notificationID}
	if n, _, err := Resolve(task); err != nil || n.Id != 0 {
		t.Fatalf("draft delivered: %+v %v", n, err)
	}
	conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 1)
	if n, _, err := Resolve(task); err != nil || n.Id == 0 {
		t.Fatalf("public source suppressed: %+v %v", n, err)
	}
	conn.Model(&posts.Entity{}).Where("id = ?", postID).Update("retention_status", posts.RetentionPurged)
	if n, _, err := Resolve(task); err != nil || n.Id != 0 {
		t.Fatalf("purged post delivered: %+v %v", n, err)
	}
}
