package sessionservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &userSessions.Entity{}); err != nil {
		t.Fatalf("migrate session test tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userSessions.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func TestCreateAndGetValidByJti(t *testing.T) {
	setupTestDB(t)

	if err := Create(100, "jti-abc", "Mozilla/5.0", "203.0.113.7"); err != nil {
		t.Fatal(err)
	}
	entity := GetValidByJti("jti-abc")
	if entity == nil {
		t.Fatal("expected valid session")
	}
	if entity.UserId != 100 || entity.Ip != "203.0.113.7" {
		t.Fatalf("session = %+v", entity)
	}
	if entity.ExpiresAt.Before(time.Now()) {
		t.Fatal("session should not be expired yet")
	}

	if got := GetValidByJti("missing"); got != nil {
		t.Fatalf("missing jti should be nil, got %+v", got)
	}
}

// TestRevokeByIDAndRevokeAll exercises single-session and all-session
// revocation. RevokeAllAndInvalidate additionally bumps token_version and
// clears the user-info cache; the transaction path is covered by the
// rollback test below.
func TestRevokeByIDAndRevokeAll(t *testing.T) {
	setupTestDB(t)

	user := users.MakeUser("session-revoke-both", "password", "session-revoke-both@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	other := users.MakeUser("session-revoke-other", "password", "session-revoke-other@example.com")
	if err := users.Create(other); err != nil {
		t.Fatalf("create other user: %v", err)
	}
	_ = Create(user.Id, "jti-1", "ua", "1.1.1.1")
	_ = Create(user.Id, "jti-2", "ua", "2.2.2.2")
	_ = Create(other.Id, "jti-3", "ua", "3.3.3.3")

	list, err := List(user.Id)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %d sessions, err = %v", len(list), err)
	}

	// Revoke one session of the first user.
	if err := RevokeByID(user.Id, list[0].Id); err != nil {
		t.Fatal(err)
	}
	if GetValidByJti(list[0].Jti) != nil {
		t.Fatal("revoked session should be gone")
	}
	// Other user's session untouched.
	if GetValidByJti("jti-3") == nil {
		t.Fatal("other user session should remain")
	}

	// Revoke all for the first user.
	if err := RevokeAllAndInvalidate(user.Id); err != nil {
		t.Fatal(err)
	}
	if GetValidByJti("jti-2") != nil {
		t.Fatal("revoke-all should remove every session of the user")
	}
}

func TestListExcludesExpiredSessions(t *testing.T) {
	setupTestDB(t)

	if err := Create(100, "jti-live", "ua", "1.1.1.1"); err != nil {
		t.Fatalf("create live session: %v", err)
	}
	if err := Create(100, "jti-expired-list", "ua", "2.2.2.2"); err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	if err := userSessions.UpdateExpiresAtByJti("jti-expired-list", time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("expire session: %v", err)
	}

	entities, err := List(100)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(entities) != 1 || entities[0].Jti != "jti-live" {
		t.Fatalf("List() = %+v, want only the live session", entities)
	}
}

// The two trigger-based tests below create and drop SQLite triggers on the
// shared :memory: test database. They must not run in parallel with any other
// test in this package, because a trigger applies to every statement on the
// shared connection.
func TestRevokeAllAndInvalidateInvalidatesCacheAfterCommit(t *testing.T) {
	setupTestDB(t)

	user := users.MakeUser("session-revoke-all", "password", "session-revoke-all@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := Create(user.Id, "jti-revoke-all", "ua", "1.1.1.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if cached, ok := userservice.GetUserInfo(user.Id); !ok || cached.TokenVersion != user.TokenVersion {
		t.Fatalf("warm user-info cache = %+v, ok = %t", cached, ok)
	}

	if err := RevokeAllAndInvalidate(user.Id); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	if GetValidByJti("jti-revoke-all") != nil {
		t.Fatal("revoke-all should delete every session")
	}
	if cached, ok := userservice.GetUserInfo(user.Id); !ok || cached.TokenVersion != user.TokenVersion+1 {
		t.Fatalf("cached token version = %d, ok = %t, want %d after commit", cached.TokenVersion, ok, user.TokenVersion+1)
	}
}

func TestRevokeAllAndInvalidateRollsBackWhenTokenVersionUpdateFails(t *testing.T) {
	setupTestDB(t)
	if !db.IsSqlite() {
		t.Skip("requires SQLite trigger support")
	}

	user := users.MakeUser("session-revoke-all-rollback", "password", "session-revoke-all-rollback@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := Create(user.Id, "jti-revoke-all-rollback", "ua", "1.1.1.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	conn := db.Connect()
	if err := conn.Exec("CREATE TRIGGER session_revoke_all_rollback BEFORE UPDATE OF token_version ON users BEGIN SELECT RAISE(ABORT, 'forced token version failure'); END;").Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Exec("DROP TRIGGER IF EXISTS session_revoke_all_rollback").Error; err != nil {
			t.Errorf("drop failure trigger: %v", err)
		}
	})

	if err := RevokeAllAndInvalidate(user.Id); err == nil {
		t.Fatal("revoke all succeeded, want token-version failure")
	}
	if GetValidByJti("jti-revoke-all-rollback") == nil {
		t.Fatal("session delete committed even though token-version update failed")
	}
	reloaded, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.TokenVersion != user.TokenVersion {
		t.Fatalf("token version = %d, want %d after rollback", reloaded.TokenVersion, user.TokenVersion)
	}
}

func TestTouchExpiry(t *testing.T) {
	setupTestDB(t)

	_ = Create(100, "jti-touch", "ua", "1.1.1.1")
	future := time.Now().Add(40 * 24 * time.Hour).Truncate(time.Second)
	persisted, err := TouchExpiry("jti-touch", future)
	if err != nil || !persisted {
		t.Fatalf("TouchExpiry() = (%t, %v), want (true, nil)", persisted, err)
	}
	entity := GetValidByJti("jti-touch")
	if entity == nil {
		t.Fatal("session should still exist")
	}
	if !entity.ExpiresAt.Equal(future) {
		t.Fatalf("expiresAt = %v, want %v", entity.ExpiresAt, future)
	}
}

func TestTouchExpiryIsMonotonicAndRejectsMissingOrExpired(t *testing.T) {
	setupTestDB(t)

	if err := Create(100, "jti-mono", "ua", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(45 * 24 * time.Hour).Truncate(time.Second)
	earlier := time.Now().Add(35 * 24 * time.Hour).Truncate(time.Second)

	persisted, err := TouchExpiry("jti-mono", later)
	if err != nil || !persisted {
		t.Fatalf("TouchExpiry(later) = (%t, %v), want (true, nil)", persisted, err)
	}
	// Out-of-order earlier expiry must not shorten the persisted deadline, while
	// still reporting persisted=true because the live row already covers earlier.
	persisted, err = TouchExpiry("jti-mono", earlier)
	if err != nil || !persisted {
		t.Fatalf("TouchExpiry(earlier) = (%t, %v), want (true, nil)", persisted, err)
	}
	entity := GetValidByJti("jti-mono")
	if entity == nil || !entity.ExpiresAt.Equal(later) {
		t.Fatalf("out-of-order write shortened expiresAt: got %+v, want %v", entity, later)
	}

	// Missing/revoked jti must report persisted=false.
	persisted, err = TouchExpiry("jti-missing", later)
	if err != nil || persisted {
		t.Fatalf("TouchExpiry(missing) = (%t, %v), want (false, nil)", persisted, err)
	}
}

func TestTouchExpiryDoesNotResurrectExpiredSession(t *testing.T) {
	setupTestDB(t)

	if err := Create(100, "jti-dead", "ua", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if err := userSessions.UpdateExpiresAtByJti("jti-dead", time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	persisted, err := TouchExpiry("jti-dead", time.Now().Add(30*24*time.Hour))
	if err != nil || persisted {
		t.Fatalf("TouchExpiry(expired) = (%t, %v), want (false, nil)", persisted, err)
	}
	if GetValidByJti("jti-dead") != nil {
		t.Fatal("TouchExpiry must not resurrect an already-expired session")
	}
}

func TestCleanupExpired(t *testing.T) {
	setupTestDB(t)

	prevMax := preferences.GetInt64("jwtopt.maxSessionLifetime", int64(jwtopt.DefaultMaxSessionLifetime/time.Second))
	preferences.Set("jwtopt.maxSessionLifetime", int64(180*86400))
	t.Cleanup(func() { preferences.Set("jwtopt.maxSessionLifetime", prevMax) })

	if err := Create(100, "jti-expired", "ua", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if err := Create(100, "jti-over-cap", "ua", "1.1.1.2"); err != nil {
		t.Fatal(err)
	}
	if err := Create(100, "jti-live-cap", "ua", "1.1.1.3"); err != nil {
		t.Fatal(err)
	}
	// Force idle expiry into the past.
	if err := userSessions.UpdateExpiresAtByJti("jti-expired", time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	// Force created_at beyond maxSessionLifetime while expires_at is still in the future.
	if err := db.Connect().Exec("UPDATE user_sessions SET created_at = ? WHERE jti = ?",
		time.Now().Add(-181*24*time.Hour), "jti-over-cap").Error; err != nil {
		t.Fatal(err)
	}

	if GetValidByJti("jti-over-cap") != nil {
		t.Fatal("session past maxSessionLifetime must not be returned as valid")
	}
	list, err := List(100)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 || list[0].Jti != "jti-live-cap" {
		t.Fatalf("List() = %+v, want only jti-live-cap", list)
	}

	CleanupExpired()
	if userSessions.GetByJti("jti-expired") != nil {
		t.Fatal("idle-expired session should have been cleaned up")
	}
	if userSessions.GetByJti("jti-over-cap") != nil {
		t.Fatal("max-lifetime-expired session should have been cleaned up")
	}
	if GetValidByJti("jti-live-cap") == nil {
		t.Fatal("live session should survive cleanup")
	}
}

func TestMaskIP(t *testing.T) {
	cases := map[string]string{
		"203.0.113.7":    "203.0.113.*",
		"203.0.113.7:80": "203.0.113.*",
		"2001:db8::1":    "2001:db8:0:0:*",
		"not-an-ip":      "",
		"":               "",
	}
	for in, want := range cases {
		if got := MaskIP(in); got != want {
			t.Fatalf("MaskIP(%q) = %q, want %q", in, got, want)
		}
	}
}
