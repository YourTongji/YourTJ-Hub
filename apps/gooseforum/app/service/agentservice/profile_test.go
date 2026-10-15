package agentservice

import (
	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"strings"
	"testing"
)

func TestManagedProfileUpdatesPreserveFieldsAndCheckLiveActor(t *testing.T) {
	setupAgentTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	owner := users.EntityComplete{Username: "profile-owner", IsActivated: users.ActivationSuccess}
	admin := users.EntityComplete{Username: "profile-admin", RoleId: 1, IsActivated: users.ActivationSuccess}
	stranger := users.EntityComplete{Username: "profile-stranger", IsActivated: users.ActivationSuccess}
	for _, u := range []*users.EntityComplete{&owner, &admin, &stranger} {
		if err := conn.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	result, err := Create(CreateParams{Username: "profile-bot", Nickname: "Original", CreatedBy: owner.Id})
	if err != nil {
		t.Fatal(err)
	}
	bio, cover, avatar := "New bio", "https://example.com/cover.png", "/static/pic/4.webp"
	view, err := Update(result.User.Id, UpdateParams{ActorID: owner.Id, Bio: &bio, ProfileCoverUrl: &cover, AvatarUrl: &avatar})
	if err != nil {
		t.Fatal(err)
	}
	if view.User.Bio != bio || view.User.Nickname != "Original" || view.User.ProfileCoverUrl != cover || view.User.AvatarUrl != avatar {
		t.Fatalf("unexpected profile: %+v", view.User)
	}
	empty := ""
	if _, err := Update(result.User.Id, UpdateParams{ActorID: admin.Id, Bio: &empty}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{stranger.Id, result.User.Id} {
		if _, err := Update(result.User.Id, UpdateParams{ActorID: id, Bio: &bio}); !errors.Is(err, ErrManagementDenied) {
			t.Fatalf("actor %d: %v", id, err)
		}
	}
	if err := conn.Model(&owner).Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := Update(result.User.Id, UpdateParams{ActorID: owner.Id, Bio: &bio}); !errors.Is(err, ErrManagementDenied) {
		t.Fatalf("frozen actor: %v", err)
	}
	for _, raw := range []string{"javascript:alert(1)", "data:image/png;base64,abc", "//example.com/image"} {
		if _, err := Update(result.User.Id, UpdateParams{ActorID: admin.Id, ProfileCoverUrl: &raw}); !errors.Is(err, ErrAgentNicknameInvalid) {
			t.Fatalf("unsafe cover %q: %v", raw, err)
		}
	}
}

func TestManagedProfileRejectsOversizedFields(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "profile-length-bot", Nickname: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		size  int
		field func(*string) UpdateParams
	}{
		{"bio", 501, func(s *string) UpdateParams { return UpdateParams{Bio: s} }},
		{"signature", 256, func(s *string) UpdateParams { return UpdateParams{Signature: s} }},
		{"website", 256, func(s *string) UpdateParams { return UpdateParams{Website: s} }},
		{"websiteName", 65, func(s *string) UpdateParams { return UpdateParams{WebsiteName: s} }},
		{"cover", 513, func(s *string) UpdateParams { return UpdateParams{ProfileCoverUrl: s} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := strings.Repeat("界", tc.size)
			if _, err := Update(result.User.Id, tc.field(&value)); !errors.Is(err, ErrAgentNicknameInvalid) {
				t.Fatalf("oversized accepted: %v", err)
			}
		})
	}
	view, err := Get(result.User.Id)
	if err != nil {
		t.Fatal(err)
	}
	if view.User.Bio != "" || view.User.Signature != "" || view.User.Website != "" || view.User.WebsiteName != "" || view.User.ProfileCoverUrl != "" {
		t.Fatal("oversized data persisted")
	}
}
