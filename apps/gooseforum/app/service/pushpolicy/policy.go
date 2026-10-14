package pushpolicy

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

const Notification = "notification"
const ChatMessage = "chat_message"

type Preferences struct {
	SystemDefaultOnly  bool `json:"systemDefaultOnly"`
	Chat               bool `json:"chat"`
	MentionsAndReplies bool `json:"mentionsAndReplies"`
	Activity           bool `json:"activity"`
	Following          bool `json:"following"`
	System             bool `json:"system"`
}
type Changes struct {
	Chat               *bool `json:"chat"`
	MentionsAndReplies *bool `json:"mentionsAndReplies"`
	Activity           *bool `json:"activity"`
	Following          *bool `json:"following"`
	System             *bool `json:"system"`
}

func Defaults() Preferences {
	return Preferences{Chat: true, MentionsAndReplies: true, System: true, SystemDefaultOnly: true}
}
func load(userID uint64) (Preferences, map[string]bool, error) {
	overrides, err := users.GetPushPreferences(userID)
	if err != nil {
		return Preferences{}, nil, err
	}
	result := Defaults()
	_, explicit := overrides["system"]
	result.SystemDefaultOnly = !explicit
	raw, err := json.Marshal(overrides)
	if err != nil {
		return Preferences{}, nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return Preferences{}, nil, err
	}
	return result, overrides, nil
}
func Get(userID uint64) (Preferences, error) { p, _, err := load(userID); return p, err }
func Update(userID uint64, changes Changes) (Preferences, error) {
	err := users.PatchPushPreferences(userID, map[string]*bool{"chat": changes.Chat, "mentionsAndReplies": changes.MentionsAndReplies, "activity": changes.Activity, "following": changes.Following, "system": changes.System})
	if err != nil {
		return Preferences{}, err
	}
	return Get(userID)
}
func Group(eventType string) string {
	switch eventType {
	case "inboxmail.delivery":
		return "inboxmail"
	case "chat_message":
		return "chat"
	case "mention", "post_reply", "comment":
		return "mentionsAndReplies"
	case "like":
		return "activity"
	case "follow", "topic_post", "wiki_updated":
		return "following"
	case "system", "badge", "review_pending", "review_approved", "review_rejected":
		return "system"
	default:
		return "unknown"
	}
}
func Allowed(userID uint64, eventType string) (bool, error) {
	p, overrides, err := load(userID)
	if err != nil {
		return false, err
	}
	switch Group(eventType) {
	case "inboxmail", "unknown":
		return false, nil
	case "chat":
		return p.Chat, nil
	case "mentionsAndReplies":
		return p.MentionsAndReplies, nil
	case "activity":
		return p.Activity, nil
	case "following":
		return p.Following, nil
	default:
		if _, explicit := overrides["system"]; !explicit {
			return eventType == "review_approved" || eventType == "review_rejected", nil
		}
		return p.System, nil
	}
}

type Task struct {
	UserId         uint64 `json:"userId"`
	NotificationId uint64 `json:"notificationId,omitempty"`
	SourceKind     string `json:"sourceKind,omitempty"`
	SourceId       uint64 `json:"sourceId,omitempty"`
}

func (t Task) Source() (string, uint64) {
	if t.SourceKind == "" {
		return Notification, t.NotificationId
	}
	return t.SourceKind, t.SourceId
}
func Enqueue(tx *gorm.DB, channel string, userID, id uint64, kind string) error {
	if userID == 0 || id == 0 {
		return nil
	}
	data := Task{UserId: userID, SourceKind: kind, SourceId: id}
	if kind == Notification {
		data.NotificationId = id
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%s:%d:%d", channel, kind, id, userID)
	err = taskQueue.CreateDeduplicatedTx(tx, &taskQueue.Entity{Type: channel + fmt.Sprint(id), TaskJson: string(raw), DedupKey: &key})
	if err == nil {
		Trace(channel, "outbox_staged", data)
	}
	return err
}
func Resolve(t Task) (eventNotification.Entity, *messages.Entity, error) {
	kind, id := t.Source()
	if t.UserId == 0 || id == 0 || users.IsAccountClosed(t.UserId) {
		return eventNotification.Entity{}, nil, nil
	}
	if kind == ChatMessage {
		msg, err := messages.GetPushSource(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return eventNotification.Entity{}, nil, nil
		}
		if err != nil {
			return eventNotification.Entity{}, nil, err
		}
		member, err := imUserChatConfigs.PushRecipientAllowed(t.UserId, msg.SenderId, msg.ConvId)
		if err != nil {
			return eventNotification.Entity{}, nil, err
		}
		if !member || msg.IsRead != 0 || msg.SenderId == t.UserId {
			return eventNotification.Entity{}, nil, nil
		}
		allowed, err := Allowed(t.UserId, ChatMessage)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return eventNotification.Entity{}, nil, nil
		}
		if !allowed && err == nil {
			Trace("policy", "user_disabled", t)
		}
		if err != nil || !allowed {
			return eventNotification.Entity{}, nil, err
		}
		blocked, err := users.InteractionBlocked(t.UserId, msg.SenderId)
		if err != nil || blocked {
			return eventNotification.Entity{}, nil, err
		}
		return eventNotification.Entity{}, &msg, nil
	}
	if kind != Notification {
		return eventNotification.Entity{}, nil, nil
	}
	n := eventNotification.GetByID(id)
	if n.Id == 0 || n.UserId != t.UserId || n.IsRead {
		return eventNotification.Entity{}, nil, nil
	}
	// Only publicly visible topic/post names can enter third-party payloads.
	if n.Payload.TopicId != 0 && n.EventType != "review_pending" && n.EventType != "review_rejected" {
		topic, err := topics.GetPushSource(n.Payload.TopicId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return eventNotification.Entity{}, nil, nil
		}
		if err != nil {
			return eventNotification.Entity{}, nil, err
		}
		n.Payload.TopicTitle = topic.Title
		if n.Payload.PostId != 0 {
			visible, err := posts.PushSourceVisible(n.Payload.PostId, n.Payload.TopicId)
			if !visible && err == nil {
				return eventNotification.Entity{}, nil, nil
			}
			if err != nil {
				return eventNotification.Entity{}, nil, err
			}
		}
	}

	actor := n.PrivateActorID
	if actor == 0 {
		actor = n.Payload.ActorId
	}
	blocked, err := users.InteractionBlocked(t.UserId, actor)
	if err != nil || blocked {
		return eventNotification.Entity{}, nil, err
	}
	allowed, err := Allowed(t.UserId, n.EventType)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return eventNotification.Entity{}, nil, nil
	}
	if !allowed && err == nil {
		Trace("policy", "user_disabled", t)
	}
	if err != nil || !allowed {
		return eventNotification.Entity{}, nil, err
	}
	return n, nil, nil
}
func Trace(channel, reason string, t Task) {
	kind, id := t.Source()
	slog.Debug("push delivery", "channel", channel, "reason", reason, "sourceKind", kind, "sourceId", id, "recipientId", t.UserId)
}

// Throttle only low-value hints; persisted notifications and chat are untouched.
func Throttle(task *taskQueue.Entity, t Task, n eventNotification.Entity, channel string) (bool, error) {
	if n.EventType != "like" && n.EventType != "topic_post" && n.EventType != "wiki_updated" && n.EventType != "follow" {
		return false, nil
	}
	target := n.Payload.TopicId
	if target == 0 {
		target = n.Payload.ActorId
	}
	if target == 0 {
		target = n.Payload.PostId
	}
	key := fmt.Sprintf("%s:%d:%s:%d", channel, t.UserId, n.EventType, target)
	blocked, err := taskQueue.ThrottlePush(task.Id, key, time.Now().Add(-30*time.Second))
	if blocked {
		Trace(channel, "throttled", t)
	}
	return blocked, err
}
