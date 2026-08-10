package eventNotification

import (
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
)

func TestClearPreviewsByTopicBlanksPayloadAndStaysPortable(t *testing.T) {
	if err := db.Connect().AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate event notification: %v", err)
	}

	// 目标通知：topicId + postId 匹配，预览与正文都应被清空。
	target := Entity{
		UserId:    1,
		EventType: EventTypeComment,
		Payload: NotificationPayload{
			Title:   "someone replied",
			Content: "sensitive body",
			TopicId: 1001,
			PostId:  2001,
			TemplateParams: NotificationTemplateParams{
				Preview: "sensitive preview",
			},
		},
	}
	if err := Create(&target); err != nil {
		t.Fatalf("create target notification: %v", err)
	}
	// 无关通知：postId 不匹配，不应被改动。
	unrelated := Entity{
		UserId:    1,
		EventType: EventTypeComment,
		Payload: NotificationPayload{
			Title:   "another reply",
			Content: "keep me",
			TopicId: 1001,
			PostId:  3001,
			TemplateParams: NotificationTemplateParams{
				Preview: "keep me preview",
			},
		},
	}
	if err := Create(&unrelated); err != nil {
		t.Fatalf("create unrelated notification: %v", err)
	}

	if err := ClearPreviewsByTopic(1001, 2001); err != nil {
		t.Fatalf("ClearPreviewsByTopic: %v", err)
	}

	reloadedTarget := findByID(t, target.Id)
	if reloadedTarget.Id == 0 {
		t.Fatal("target notification missing after clear")
	}
	if reloadedTarget.Payload.TemplateParams.Preview != "" {
		t.Fatalf("target preview not cleared: %q", reloadedTarget.Payload.TemplateParams.Preview)
	}
	if reloadedTarget.Payload.Content != "" {
		t.Fatalf("target content not cleared: %q", reloadedTarget.Payload.Content)
	}
	if reloadedTarget.Payload.TopicId != 1001 || reloadedTarget.Payload.PostId != 2001 {
		t.Fatalf("target identity fields lost: %#v", reloadedTarget.Payload)
	}

	reloadedUnrelated := findByID(t, unrelated.Id)
	if reloadedUnrelated.Id == 0 {
		t.Fatal("unrelated notification missing after clear")
	}
	if reloadedUnrelated.Payload.TemplateParams.Preview != "keep me preview" {
		t.Fatalf("unrelated preview was modified: %q", reloadedUnrelated.Payload.TemplateParams.Preview)
	}
}

func TestClearPreviewsByTopicByTopicOnly(t *testing.T) {
	if err := db.Connect().AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate event notification: %v", err)
	}

	first := Entity{
		UserId:    1,
		EventType: EventTypeTopicPost,
		Payload: NotificationPayload{
			Title:   "post in followed topic",
			Content: "preview body",
			TopicId: 2001,
			PostId:  9001,
			TemplateParams: NotificationTemplateParams{
				Preview: "preview text",
			},
		},
	}
	if err := Create(&first); err != nil {
		t.Fatalf("create notification: %v", err)
	}

	// postId=0 表示清空整个话题的所有通知预览。
	if err := ClearPreviewsByTopic(2001, 0); err != nil {
		t.Fatalf("ClearPreviewsByTopic(topic only): %v", err)
	}

	reloaded := findByID(t, first.Id)
	if reloaded.Id == 0 {
		t.Fatal("notification missing after clear")
	}
	if reloaded.Payload.TemplateParams.Preview != "" || reloaded.Payload.Content != "" {
		t.Fatalf("topic-level preview not cleared: %#v", reloaded.Payload)
	}
}

// findByID 通过 QueryByUserId 分页读取定位通知，验证 payload 在重新序列化后仍可解析。
func findByID(t *testing.T, id uint64) Entity {
	t.Helper()
	notifications, err := QueryByUserId(1, 100, 0, false)
	if err != nil {
		t.Fatalf("QueryByUserId: %v", err)
	}
	for _, item := range notifications {
		if item.Id == id {
			return *item
		}
	}
	return Entity{}
}
