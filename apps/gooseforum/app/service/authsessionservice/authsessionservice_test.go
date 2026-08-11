package authsessionservice

import (
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
)

func setupAuthSessionTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &userSessions.Entity{}); err != nil {
		t.Fatalf("migrate auth session test tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userSessions.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

// TestValidateTokenAfterRevokeAll pins the end-to-end token-version chain that
// the auth middleware relies on: after RevokeAllAndInvalidate, an old token
// must be rejected (session row gone + TokenVersion bumped) and a freshly
// issued token must be accepted even though the user-info cache may have
// served a stale TokenVersion between the commit and the invalidation.
func TestValidateTokenAfterRevokeAll(t *testing.T) {
	setupAuthSessionTestDB(t)

	user := users.MakeUser("auth-validate-revoke", "password", "auth-validate-revoke@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken: %v", err)
	}
	if err := sessionservice.Create(user.Id, jti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, _, _, ok := ValidateToken(token); !ok {
		t.Fatal("ValidateToken must accept a live session token")
	}

	if err := sessionservice.RevokeAllAndInvalidate(user.Id); err != nil {
		t.Fatalf("RevokeAllAndInvalidate: %v", err)
	}
	if _, _, _, ok := ValidateToken(token); ok {
		t.Fatal("ValidateToken must reject a token after revoke-all")
	}

	reloaded, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	newToken, newJti, err := jwtopt.CreateSessionToken(user.Id, reloaded.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken after revoke-all: %v", err)
	}
	if err := sessionservice.Create(user.Id, newJti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("recreate session: %v", err)
	}
	if _, _, _, ok := ValidateToken(newToken); !ok {
		t.Fatal("ValidateToken must accept a freshly issued token after revoke-all (cache must not serve the stale TokenVersion)")
	}
}

// TestValidateTokenRejectsBot mirrors the middleware contract that bot
// accounts can never authenticate through the session-token path.
func TestValidateTokenRejectsBot(t *testing.T) {
	setupAuthSessionTestDB(t)

	bot := users.EntityComplete{
		Username:    "auth-validate-bot",
		ActorType:   users.ActorTypeBot,
		IsActivated: users.ActivationSuccess,
	}
	if err := db.Connect().Create(&bot).Error; err != nil {
		t.Fatalf("create bot: %v", err)
	}
	token, jti, err := jwtopt.CreateSessionToken(bot.Id, bot.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken: %v", err)
	}
	if err := sessionservice.Create(bot.Id, jti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, _, _, ok := ValidateToken(token); ok {
		t.Fatal("ValidateToken must reject bot accounts")
	}
}
