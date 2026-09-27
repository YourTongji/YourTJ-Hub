package authsessionservice

import (
	"context"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
)

func TestCheckStreamTokenSeesRevocationAndDistinguishesDatabaseFailure(t *testing.T) {
	setupAuthSessionTestDB(t)
	user := users.MakeUser("stream-auth", "password", "stream-auth@example.com")
	if err := users.Create(user); err != nil {
		t.Fatal(err)
	}
	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, jti, "test", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if valid, err := CheckStreamToken(context.Background(), token); err != nil || !valid {
		t.Fatalf("live token: valid=%t err=%v", valid, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CheckStreamToken(ctx, token); err == nil {
		t.Fatal("database cancellation must not be classified as revoked")
	}
	if err := db.Connect().Model(&users.EntityComplete{}).Where("id = ?", user.Id).
		Update("token_version", user.TokenVersion+1).Error; err != nil {
		t.Fatal(err)
	}
	if valid, err := CheckStreamToken(context.Background(), token); err != nil || valid {
		t.Fatalf("rotated token version: valid=%t err=%v", valid, err)
	}
	if err := db.Connect().Model(&users.EntityComplete{}).Where("id = ?", user.Id).
		Update("token_version", user.TokenVersion).Error; err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.RevokeByJti(user.Id, jti); err != nil {
		t.Fatal(err)
	}
	if valid, err := CheckStreamToken(context.Background(), token); err != nil || valid {
		t.Fatalf("revoked token: valid=%t err=%v", valid, err)
	}
}
