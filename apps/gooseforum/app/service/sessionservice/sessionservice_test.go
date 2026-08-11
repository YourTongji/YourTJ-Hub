package sessionservice

import (
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/userservice"
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

func TestRevokeByIDAndRevokeAll(t *testing.T) {
	setupTestDB(t)

	_ = Create(100, "jti-1", "ua", "1.1.1.1")
	_ = Create(100, "jti-2", "ua", "2.2.2.2")
	_ = Create(200, "jti-3", "ua", "3.3.3.3")

	list, err := List(100)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %d sessions, err = %v", len(list), err)
	}

	// Revoke one session of user 100.
	if err := RevokeByID(100, list[0].Id); err != nil {
		t.Fatal(err)
	}
	if GetValidByJti(list[0].Jti) != nil {
		t.Fatal("revoked session should be gone")
	}
	// Other user's session untouched.
	if GetValidByJti("jti-3") == nil {
		t.Fatal("other user session should remain")
	}

	// Revoke all for user 100.
	if err := RevokeAll(100); err != nil {
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
	future := time.Now().Add(30 * 24 * time.Hour)
	TouchExpiry("jti-touch", future)
	entity := GetValidByJti("jti-touch")
	if entity == nil {
		t.Fatal("session should still exist")
	}
	if !entity.ExpiresAt.Equal(future) {
		t.Fatalf("expiresAt = %v, want %v", entity.ExpiresAt, future)
	}
}

func TestCleanupExpired(t *testing.T) {
	setupTestDB(t)

	if err := Create(100, "jti-expired", "ua", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	// Force expiry into the past.
	if err := userSessions.UpdateExpiresAtByJti("jti-expired", time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	CleanupExpired()
	if GetValidByJti("jti-expired") != nil {
		t.Fatal("expired session should have been cleaned up")
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
