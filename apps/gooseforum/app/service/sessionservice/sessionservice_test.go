package sessionservice

import (
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&userSessions.Entity{}); err != nil {
		t.Fatalf("migrate user_sessions: %v", err)
	}
	conn.Where("1 = 1").Delete(&userSessions.Entity{})
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
		"":               "",
	}
	for in, want := range cases {
		if got := MaskIP(in); got != want {
			t.Fatalf("MaskIP(%q) = %q, want %q", in, got, want)
		}
	}
}
