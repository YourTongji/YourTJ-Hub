package webpushservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"strings"
	"testing"
)

func TestSafeCopyLanguagesAndDualReason(t *testing.T) {
	for _, lang := range []string{"zh", "en", "ja", "de"} {
		for _, typ := range []string{"review_pending", "review_rejected"} {
			copy := BuildPushCopy(eventNotification.Entity{EventType: typ, Payload: eventNotification.NotificationPayload{TopicTitle: "SECRET CANDIDATE", Content: "SECRET BODY"}}, lang)
			if copy == nil || strings.Contains(copy.Title+copy.Body, "SECRET") {
				t.Fatalf("private review copy %s/%s %+v", lang, typ, copy)
			}
		}
		n := eventNotification.Entity{EventType: "post_reply", Payload: eventNotification.NotificationPayload{ActorPersonaUID: "anonymous", ActorName: "Anonymous", ActorId: 91, TopicTitle: "Public title", HasDirectMention: true}}
		copy := BuildPushCopy(n, lang)
		if copy == nil || !strings.Contains(copy.Body, "Anonymous") || !strings.Contains(copy.Body, bodyText(lang, "mention")) {
			t.Fatalf("dual reason/persona copy %s %+v", lang, copy)
		}
		chat := BuildChatCopy(0, lang)
		if chat.Title == "" || chat.Body == "" {
			t.Fatalf("missing private chat language %s", lang)
		}
	}
}

func TestBadgeWithoutMetadataDoesNotInventName(t *testing.T) {
	want := map[string]string{"zh": "你获得了一枚新徽章", "en": "You earned a new badge", "ja": "新しいバッジを獲得しました", "de": "Du hast ein neues Abzeichen erhalten"}
	for lang, body := range want {
		copy := BuildPushCopy(eventNotification.Entity{EventType: eventNotification.EventTypeBadge}, lang)
		if copy == nil || copy.Title != body || copy.Body != body {
			t.Fatalf("missing badge metadata %s: %+v; want %q", lang, copy, body)
		}
	}
}
