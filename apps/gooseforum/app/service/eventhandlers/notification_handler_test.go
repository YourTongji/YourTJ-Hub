package eventhandlers

import (
	"context"
	"strconv"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestCommentNotificationExcludeUserIds(t *testing.T) {
	event := &CommentCreatedEvent{
		UserId:              1,
		TopicAuthorId:       2,
		ReplyToPostAuthorId: 2,
	}

	userIds := commentNotificationExcludeUserIds(event)
	got := make(map[uint64]bool, len(userIds))
	for _, userId := range userIds {
		got[userId] = true
	}

	if len(got) != 2 || !got[1] || !got[2] {
		t.Fatalf("unexpected exclude user ids: %#v", userIds)
	}
}

func TestCommentCreatedEventCarriesTopicPostPosition(t *testing.T) {
	event := &CommentCreatedEvent{TopicId: 30, PostId: 40, PostNo: 5}
	if event.TopicId != 30 || event.PostId != 40 || event.PostNo != 5 {
		t.Fatalf("event position = %#v", event)
	}
}

func TestPostLikedEventCarriesTopicPostPosition(t *testing.T) {
	event := &PostLikedEvent{UserId: 1, PostId: 40, PostNo: 5, TopicId: 30, LikerId: 2}
	if event.PostId != 40 || event.PostNo != 5 {
		t.Fatalf("event position = %#v", event)
	}
}

func TestShouldNotifyTopicAuthor(t *testing.T) {
	tests := []struct {
		name  string
		event *CommentCreatedEvent
		want  bool
	}{
		{
			name: "root comment notifies topic author",
			event: &CommentCreatedEvent{
				UserId:        1,
				TopicAuthorId: 2,
			},
			want: true,
		},
		{
			name: "topic author own comment does not notify",
			event: &CommentCreatedEvent{
				UserId:        2,
				TopicAuthorId: 2,
			},
			want: false,
		},
		{
			name: "reply to another user still notifies topic author",
			event: &CommentCreatedEvent{
				UserId:              1,
				TopicAuthorId:       2,
				ReplyToPostId:       10,
				ReplyToPostAuthorId: 3,
			},
			want: true,
		},
		{
			name: "reply to topic author only sends reply notification",
			event: &CommentCreatedEvent{
				UserId:              1,
				TopicAuthorId:       2,
				ReplyToPostId:       10,
				ReplyToPostAuthorId: 2,
			},
			want: false,
		},
		{
			name: "missing topic author does not notify",
			event: &CommentCreatedEvent{
				UserId: 1,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldNotifyTopicAuthor(tt.event); got != tt.want {
				t.Fatalf("shouldNotifyTopicAuthor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldNotifyParentReplyAuthor(t *testing.T) {
	tests := []struct {
		name  string
		event *CommentCreatedEvent
		want  bool
	}{
		{
			name: "root comment does not notify parent reply author",
			event: &CommentCreatedEvent{
				UserId:              1,
				ReplyToPostAuthorId: 2,
			},
			want: false,
		},
		{
			name: "reply notifies parent reply author",
			event: &CommentCreatedEvent{
				UserId:              1,
				ReplyToPostId:       10,
				ReplyToPostAuthorId: 2,
			},
			want: true,
		},
		{
			name: "self reply does not notify",
			event: &CommentCreatedEvent{
				UserId:              1,
				ReplyToPostId:       10,
				ReplyToPostAuthorId: 1,
			},
			want: false,
		},
		{
			name: "missing parent reply author does not notify",
			event: &CommentCreatedEvent{
				UserId:        1,
				ReplyToPostId: 10,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldNotifyParentReplyAuthor(tt.event); got != tt.want {
				t.Fatalf("shouldNotifyParentReplyAuthor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTakeUpTo64Chars(t *testing.T) {
	short := "短内容"
	if got := TakeUpTo64Chars(short); got != short {
		t.Fatalf("short content changed: %q", got)
	}

	var long strings.Builder
	for range 70 {
		long.WriteString("鹅")
	}
	got := TakeUpTo64Chars(long.String())
	if len([]rune(got)) != 64 {
		t.Fatalf("TakeUpTo64Chars() length = %d, want 64", len([]rune(got)))
	}
}

func TestTakeUpTo64CharsRemovesMarkdownImageSyntax(t *testing.T) {
	got := TakeUpTo64Chars("![image](/file/img/2026/07/example.png)")
	if got != "[图片]" {
		t.Fatalf("TakeUpTo64Chars() = %q, want [图片]", got)
	}
}

func TestPriorityRecipients(t *testing.T) {
	tests := []struct {
		name           string
		event          *CommentCreatedEvent
		mentionIDs     []uint64
		wantRecipients map[uint64]string
	}{
		{
			name:           "root comment notifies topic author",
			event:          &CommentCreatedEvent{UserId: 1, TopicAuthorId: 2},
			wantRecipients: map[uint64]string{2: "comment"},
		},
		{
			name:           "reply notifies parent with post_reply",
			event:          &CommentCreatedEvent{UserId: 1, TopicAuthorId: 2, ReplyToPostId: 10, ReplyToPostAuthorId: 3},
			wantRecipients: map[uint64]string{2: "comment", 3: "post_reply"},
		},
		{
			name:           "reply to topic author only gets post_reply",
			event:          &CommentCreatedEvent{UserId: 1, TopicAuthorId: 2, ReplyToPostId: 10, ReplyToPostAuthorId: 2},
			wantRecipients: map[uint64]string{2: "post_reply"},
		},
		{
			name:           "mention beats comment",
			event:          &CommentCreatedEvent{UserId: 1, TopicAuthorId: 2},
			mentionIDs:     []uint64{2},
			wantRecipients: map[uint64]string{2: "mention"},
		},
		{
			name:           "post_reply beats mention",
			event:          &CommentCreatedEvent{UserId: 1, TopicAuthorId: 2, ReplyToPostId: 10, ReplyToPostAuthorId: 3},
			mentionIDs:     []uint64{3},
			wantRecipients: map[uint64]string{2: "comment", 3: "post_reply"},
		},
		{
			name:           "mention wins over topic watch candidate",
			event:          &CommentCreatedEvent{UserId: 1, TopicAuthorId: 2},
			mentionIDs:     []uint64{3},
			wantRecipients: map[uint64]string{2: "comment", 3: "mention"},
		},
		{
			name:           "zero ids skipped",
			event:          &CommentCreatedEvent{UserId: 1},
			mentionIDs:     []uint64{0, 2},
			wantRecipients: map[uint64]string{2: "mention"},
		},
		{
			name:           "own comment no self notification",
			event:          &CommentCreatedEvent{UserId: 2, TopicAuthorId: 2, ReplyToPostId: 10, ReplyToPostAuthorId: 2},
			wantRecipients: map[uint64]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := priorityRecipients(tt.event, tt.mentionIDs)
			if len(got) != len(tt.wantRecipients) {
				t.Fatalf("priorityRecipients() = %v, want %v", got, tt.wantRecipients)
			}
			for _, recipient := range got {
				if want, ok := tt.wantRecipients[recipient.userID]; !ok || recipient.eventType != want {
					t.Fatalf("priorityRecipients() = %v, want %v", got, tt.wantRecipients)
				}
			}
		})
	}
}

func TestMentionDiff(t *testing.T) {
	tests := []struct {
		name string
		old  []uint64
		new  []uint64
		want []uint64
	}{
		{name: "added mention notifies", old: []uint64{1}, new: []uint64{1, 2}, want: []uint64{2}},
		{name: "kept mention no re-notify", old: []uint64{1, 2}, new: []uint64{1, 2}, want: nil},
		{name: "removed mention no notify", old: []uint64{1, 2}, new: []uint64{1}, want: nil},
		{name: "text only change no notify", old: []uint64{1}, new: []uint64{1}, want: nil},
		{name: "empty old all new notify", old: nil, new: []uint64{1, 2}, want: []uint64{1, 2}},
		{name: "fanout capped at 20", old: nil, new: makeRange(1, 25), want: makeRange(1, 20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mentionDiff(tt.old, tt.new)
			if strings.Join(loMap(got), ",") != strings.Join(loMap(tt.want), ",") {
				t.Fatalf("mentionDiff(%v, %v) = %v, want %v", tt.old, tt.new, got, tt.want)
			}
		})
	}
}

func TestHandleCommentCreatedAnonymousSkipsNotifications(t *testing.T) {
	// 匿名楼层直接返回，不触达数据库（issue #524 边界，issue #563 要求保持）。
	if err := handleCommentCreated(context.Background(), &CommentCreatedEvent{IsAnonymous: true, Content: "@alice"}); err != nil {
		t.Fatalf("handleCommentCreated(anonymous) err=%v", err)
	}
}

func makeRange(from, to int) []uint64 {
	out := make([]uint64, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, uint64(i))
	}
	return out
}

func loMap(ids []uint64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatUint(id, 10))
	}
	return out
}

func TestResolveMentionUserIDs(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	alice := users.MakeUser("alice", "pass1234", "alice@example.com")
	bob := users.MakeUser("bob", "pass1234", "bob@example.com")
	if err := users.Create(alice); err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := users.Create(bob); err != nil {
		t.Fatalf("create bob: %v", err)
	}

	t.Run("resolves dedup skips unknown and self", func(t *testing.T) {
		got := resolveMentionUserIDs("你好 @alice @bob @alice @nobody", alice.Id, 0)
		if len(got) != 1 || got[0] != bob.Id {
			t.Fatalf("resolveMentionUserIDs() = %v, want [%d]", got, bob.Id)
		}
	})
	t.Run("limit caps fanout", func(t *testing.T) {
		got := resolveMentionUserIDs("@alice @bob", 0, 1)
		if len(got) != 1 {
			t.Fatalf("resolveMentionUserIDs(limit=1) = %v, want 1 id", got)
		}
	})
	t.Run("no mentions returns nil", func(t *testing.T) {
		if got := resolveMentionUserIDs("无提及", 0, 0); got != nil {
			t.Fatalf("resolveMentionUserIDs() = %v, want nil", got)
		}
	})
}
