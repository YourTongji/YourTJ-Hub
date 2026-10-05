package tokenservice

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestModerationActionTokenRoundTrip(t *testing.T) {
	withAppSigningKey(t, "moderation-action-test-key-0123456789")
	now := time.Unix(1_800_000_000, 0)
	claims := ModerationActionClaims{Subject: ModerationActionSubjectReviewPost, ID: 42, Action: "approve", Version: "abc"}
	token, err := IssueModerationAction(claims, now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseModerationAction(token, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Subject != claims.Subject || got.ID != 42 || got.Action != "approve" || got.Version != "abc" {
		t.Fatalf("claims = %+v", got)
	}
	if got.Expires != now.Add(ModerationActionTTL).Unix() {
		t.Fatalf("expires = %d", got.Expires)
	}
}

func TestModerationActionTokenRejectsTamperExpiryAndForeignKeys(t *testing.T) {
	withAppSigningKey(t, "moderation-action-test-key-0123456789")
	now := time.Unix(1_800_000_000, 0)
	token, err := IssueModerationAction(ModerationActionClaims{Subject: ModerationActionSubjectReport, ID: 7, Action: "dismiss"}, now)
	if err != nil {
		t.Fatal(err)
	}

	// 改写动作（payload 段）后签名必须失效：链接不能被改成另一个动作。
	body, sig, _ := strings.Cut(strings.TrimPrefix(token, "v1."), ".")
	forged, err := IssueModerationAction(ModerationActionClaims{Subject: ModerationActionSubjectReport, ID: 7, Action: "ban"}, now)
	if err != nil {
		t.Fatal(err)
	}
	forgedBody, _, _ := strings.Cut(strings.TrimPrefix(forged, "v1."), ".")
	for _, bad := range []string{
		"", "v1.", "v2." + body + "." + sig, "v1." + forgedBody + "." + sig, "v1." + body + "." + sig + "x", token + "x",
	} {
		if _, err := ParseModerationAction(bad, now); !errors.Is(err, ErrModerationActionInvalid) {
			t.Fatalf("token %q: err = %v, want invalid", bad, err)
		}
	}

	expired, err := ParseModerationAction(token, now.Add(ModerationActionTTL+time.Second))
	if !errors.Is(err, ErrModerationActionExpired) || expired.ID != 7 {
		t.Fatalf("expired: claims=%+v err=%v", expired, err)
	}

	// 轮换 signing key 后旧链接全部失效。
	withAppSigningKey(t, "rotated-moderation-action-key-987654")
	if _, err := ParseModerationAction(token, now); !errors.Is(err, ErrModerationActionInvalid) {
		t.Fatalf("rotated key accepted old token: %v", err)
	}
}

func TestModerationActionTokenIsNotAPasswordResetToken(t *testing.T) {
	withAppSigningKey(t, "moderation-action-test-key-0123456789")
	reset, err := GeneratePasswordResetToken(1, "user@example.com", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseModerationAction(reset, time.Now()); err == nil {
		t.Fatal("password reset token accepted as moderation action token")
	}
	action, err := IssueModerationAction(ModerationActionClaims{Subject: ModerationActionSubjectReport, ID: 1, Action: "dismiss"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePasswordResetToken(action); err == nil {
		t.Fatal("moderation action token accepted as password reset token")
	}
}
