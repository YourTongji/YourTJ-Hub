package users

import (
	"errors"
	"testing"
)

func TestUnreadBadgeStyleDefaultsIsolationAndValidation(t *testing.T) {
	setupUserIsolationTestDB(t)
	owner := MakeUser("badge-owner", "old-password", "badge-owner@example.test")
	other := MakeUser("badge-other", "old-password", "badge-other@example.test")
	for _, user := range []*EntityComplete{owner, other} {
		if err := Create(user); err != nil {
			t.Fatal(err)
		}
	}
	if style, err := GetUnreadBadgeStyle(owner.Id); err != nil || style != UnreadBadgeNumber {
		t.Fatalf("default = %q, %v", style, err)
	}
	if err := SetUnreadBadgeStyle(owner.Id, UnreadBadgeDot); err != nil {
		t.Fatal(err)
	}
	if style, err := GetUnreadBadgeStyle(owner.Id); err != nil || style != UnreadBadgeDot {
		t.Fatalf("saved style = %q, %v", style, err)
	}
	if style, err := GetUnreadBadgeStyle(other.Id); err != nil || style != UnreadBadgeNumber {
		t.Fatalf("another account = %q, %v", style, err)
	}
	if err := SetUnreadBadgeStyle(owner.Id, "unsupported"); !errors.Is(err, ErrUnreadBadgeStyle) {
		t.Fatalf("invalid style = %v", err)
	}
	if style, _ := GetUnreadBadgeStyle(owner.Id); style != UnreadBadgeDot {
		t.Fatal("invalid update changed the preference")
	}
	row, err := Get(owner.Id)
	if err != nil || row.Password != owner.Password || row.TokenVersion != owner.TokenVersion {
		t.Fatal("display update changed account credentials")
	}
}
