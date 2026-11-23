package authsessionservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
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

func TestValidateSessionSlidesActiveSessionAndEnforcesMaxLifetime(t *testing.T) {
	setupAuthSessionTestDB(t)

	prevValid := preferences.GetInt64("jwtopt.validTime", int64(jwtopt.DefaultValidTime/time.Second))
	prevRefresh := preferences.GetInt64("jwtopt.refreshInterval", int64(jwtopt.DefaultRefreshInterval/time.Second))
	prevMax := preferences.GetInt64("jwtopt.maxSessionLifetime", int64(jwtopt.DefaultMaxSessionLifetime/time.Second))
	preferences.Set("jwtopt.validTime", int64(30*86400))
	preferences.Set("jwtopt.refreshInterval", int64(86400))
	preferences.Set("jwtopt.maxSessionLifetime", int64(180*86400))
	t.Cleanup(func() {
		preferences.Set("jwtopt.validTime", prevValid)
		preferences.Set("jwtopt.refreshInterval", prevRefresh)
		preferences.Set("jwtopt.maxSessionLifetime", prevMax)
	})

	user := users.MakeUser("auth-slide-user", "password", "auth-slide-user@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 1. Fresh session does not refresh immediately.
	freshToken, freshJti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, freshJti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	res, ok := ValidateSession(freshToken)
	if !ok {
		t.Fatal("ValidateSession must accept fresh token")
	}
	if res.Refreshed || res.Token != freshToken {
		t.Fatal("fresh token within refreshInterval must not refresh")
	}

	// 2. Active session > 24h after issue (28d remaining) slides forward to ~30d.
	agedJti, err := jwtopt.GenerateJti()
	if err != nil {
		t.Fatal(err)
	}
	agedToken, err := jwtopt.Std().CreateToken(jwtopt.CustomClaims{
		UserId:           user.Id,
		TokenVersion:     user.TokenVersion,
		Jti:              agedJti,
		RegisteredClaims: jwtopt.GetBaseRegisteredClaims(28 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, agedJti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	res, ok = ValidateSession(agedToken)
	if !ok {
		t.Fatal("ValidateSession must accept aged token")
	}
	if !res.Refreshed || res.Token == agedToken {
		t.Fatal("aged token (>24h elapsed) must refresh")
	}
	if time.Until(res.ExpiresAt) < 29*24*time.Hour {
		t.Fatalf("refreshed expiresAt = %v, want ~30d from now", res.ExpiresAt)
	}

	// 3. Session near maxSessionLifetime (created 170d ago) clamps to createdAt + 180d.
	nearCapJti, err := jwtopt.GenerateJti()
	if err != nil {
		t.Fatal(err)
	}
	nearCapToken, err := jwtopt.Std().CreateToken(jwtopt.CustomClaims{
		UserId:           user.Id,
		TokenVersion:     user.TokenVersion,
		Jti:              nearCapJti,
		RegisteredClaims: jwtopt.GetBaseRegisteredClaims(5 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, nearCapJti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	created170dAgo := time.Now().Add(-170 * 24 * time.Hour).Truncate(time.Second)
	if err := db.Connect().Exec("UPDATE user_sessions SET created_at = ? WHERE jti = ?",
		created170dAgo, nearCapJti).Error; err != nil {
		t.Fatal(err)
	}
	res, ok = ValidateSession(nearCapToken)
	if !ok || !res.Refreshed {
		t.Fatalf("near-cap ValidateSession: ok=%t refreshed=%t", ok, res.Refreshed)
	}
	if time.Until(res.ExpiresAt) > 10*24*time.Hour+time.Minute {
		t.Fatalf("near-cap expiresAt = %v exceeded 10d remaining cap", res.ExpiresAt)
	}

	// 4. Session past maxSessionLifetime is rejected even if token/row expires_at is future.
	if err := db.Connect().Exec("UPDATE user_sessions SET created_at = ? WHERE jti = ?",
		time.Now().Add(-181*24*time.Hour), nearCapJti).Error; err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateSession(nearCapToken); ok {
		t.Fatal("ValidateSession must reject session past maxSessionLifetime")
	}
}
