package agentservice

import (
	"errors"
	"testing"

	fileDB "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
)

func TestAvatarUpdateRollsBackAfterUploadReclamationClaim(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "reclaimed-avatar", Nickname: "Reclaimed Avatar", CreatedBy: 42})
	if err != nil {
		t.Fatal(err)
	}
	if err := fileDB.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatal(err)
	}
	name := "2026/10/reclaimed-agent-avatar.webp"
	if _, err := filedata.SaveFile(result.User.Id, name, "image/webp", []byte("avatar")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fileDB.Connect().Where("name = ?", name).Delete(&filedata.Entity{}) })
	if err := fileusageservice.AddUploadOwner(result.User.Id, name); err != nil {
		t.Fatal(err)
	}
	if claimed, err := fileUsage.ClaimOrphanUpload(name); err != nil || !claimed {
		t.Fatalf("claim = %v, %v", claimed, err)
	}
	_, err = Update(result.User.Id, UpdateParams{AvatarUrl: &name})
	if !errors.Is(err, fileUsage.ErrUploadReclaimed) {
		t.Fatalf("avatar admitted after reclamation claim: %v", err)
	}
	user, err := users.Get(result.User.Id)
	if err != nil {
		t.Fatal(err)
	}
	if user.AvatarUrl != result.User.AvatarUrl {
		t.Fatal("failed avatar update changed the user")
	}
}
