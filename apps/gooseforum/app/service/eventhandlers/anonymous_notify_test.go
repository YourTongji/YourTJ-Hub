package eventhandlers

import (
	"encoding/json"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestAnonymousWebhookProjectionAndRetryNeverRestoreOwner(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&identity.Persona{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	const uid = "cdef0123456789abcdef0123456789ab"
	const owner = uint64(9894201)
	persona := identity.Persona{UID: uid, Name: "旧花名", AvatarSeed: "secret-avatar-seed"}
	if err := conn.Create(&persona).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: owner, Username: "private-webhook-owner"}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Delete(&identity.Persona{}, "uid = ?", uid)
		conn.Unscoped().Delete(&users.EntityComplete{}, owner)
	})
	topic := topics.Entity{Id: 9894202, UserId: owner, PersonaUID: uid, Title: "公开正文"}
	for _, state := range []string{"initial", "renamed", "missing"} {
		t.Run(state, func(t *testing.T) {
			if state == "renamed" {
				if err := conn.Model(&persona).Update("name", "新花名").Error; err != nil {
					t.Fatal(err)
				}
			}
			if state == "missing" {
				if err := conn.Delete(&persona).Error; err != nil {
					t.Fatal(err)
				}
			}
			payload := topicNotifyPayload(&topic)
			if payload.User.ID != 0 || payload.Topic.UserID != 0 || payload.User.PublicUID != uid || !strings.HasSuffix(payload.User.URL, "/a/"+uid) {
				t.Fatalf("private webhook author: %+v", payload)
			}
			if state == "renamed" && payload.User.DisplayName != "新花名" {
				t.Fatal("retry kept a stale name")
			}
			encoded, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"private-webhook-owner", "9894201", "secret-avatar-seed", "/u/"} {
				if strings.Contains(string(encoded), secret) {
					t.Fatalf("webhook leaked %q: %s", secret, encoded)
				}
			}
		})
	}
}
