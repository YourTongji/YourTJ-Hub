package notificationservice

import (
	"encoding/json"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestAnonymousNotificationPersistenceAndHydrationNeverRestoreOwner(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &posts.Entity{}, &topics.Entity{}, &identity.Persona{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	const owner = uint64(9892201)
	const postID = uint64(9892202)
	const uid = "abcdef0123456789abcdef0123456789"
	t.Cleanup(func() {
		conn.Where("private_actor_id = ?", owner).Delete(&eventNotification.Entity{})
		conn.Unscoped().Delete(&posts.Entity{}, postID)
		conn.Unscoped().Delete(&users.EntityComplete{}, owner)
		conn.Delete(&identity.Persona{}, "uid = ?", uid)
	})
	for _, row := range []any{&users.EntityComplete{Id: owner, Username: "private-notification-owner"}, &posts.Entity{Id: postID, PersonaUID: uid, IsAnonymous: true, UserId: owner}, &identity.Persona{UID: uid, Name: "同学", AvatarSeed: "private-avatar-seed"}} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	comment := &eventNotification.Entity{EventType: eventNotification.EventTypeComment, Payload: eventNotification.NotificationPayload{PostId: postID, ActorId: owner, ActorName: "private-notification-owner"}}
	like := &eventNotification.Entity{EventType: eventNotification.EventTypeLike, Payload: eventNotification.NotificationPayload{PostId: postID, ActorId: owner, ActorName: "private-notification-owner"}}
	if err := conn.Transaction(func(tx *gorm.DB) error { return redactAnonymousActors(tx, []*eventNotification.Entity{comment, like}) }); err != nil {
		t.Fatal(err)
	}
	if comment.PrivateActorID != owner || comment.Payload.ActorId != 0 || comment.Payload.ActorPersonaUID != uid || like.Payload.ActorId != 0 || like.Payload.ActorName != "" {
		t.Fatal("incorrect notification projection", comment, like)
	}
	if err := conn.Create(comment).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&identity.Persona{}).Where("uid = ?", uid).Update("name", "新花名").Error; err != nil {
		t.Fatal(err)
	}
	var persisted eventNotification.Entity
	if err := conn.First(&persisted, comment.Id).Error; err != nil {
		t.Fatal(err)
	}
	// A stale/old actor field must not beat the immutable public persona reference.
	persisted.Payload.ActorId = owner
	if err := hydrateNotifications([]*eventNotification.Entity{&persisted, like}); err != nil {
		t.Fatal(err)
	}
	if persisted.Payload.ActorName != "新花名" || persisted.Payload.ActorId != 0 || like.Payload.ActorName != "" {
		t.Fatal("hydration restored private actor", persisted, like)
	}
	encoded, err := json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-notification-owner", "privateActor", "private-avatar-seed"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("notification leaks %q: %s", secret, encoded)
		}
	}
	if err := conn.Delete(&identity.Persona{}, "uid = ?", uid).Error; err != nil {
		t.Fatal(err)
	}
	persisted.Payload.ActorId = owner
	if err := hydrateNotifications([]*eventNotification.Entity{&persisted}); err != nil {
		t.Fatal(err)
	}
	if persisted.Payload.ActorId != 0 || persisted.Payload.ActorName == "private-notification-owner" {
		t.Fatal("missing persona fell back to owner")
	}
}
