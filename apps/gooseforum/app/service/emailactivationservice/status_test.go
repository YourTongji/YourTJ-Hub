package emailactivationservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"testing"
	"time"
)

func TestActivationStatusEligibility(t *testing.T) {
	now := time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)
	user := users.EntityComplete{Email: "current@example.test", IsActivated: users.ActivationPending}
	cases := []struct {
		name                string
		user                users.EntityComplete
		enabled, mailReady  bool
		state, kind, target string
	}{
		{"pending", user, true, true, "ready", "current", user.Email},
		{"disabled", user, false, true, "disabled", "current", user.Email},
		{"mail unavailable", user, true, false, "mail_unavailable", "current", user.Email},
		{"missing email", users.EntityComplete{}, true, true, "missing_email", "none", ""},
		{"verified", users.EntityComplete{IsActivated: 1, Email: user.Email}, true, true, "verified", "none", ""},
		{"frozen", users.EntityComplete{IsFrozen: users.StatusFrozen}, false, false, "frozen", "none", ""},
		{"governance blocked", users.EntityComplete{AnonymousGovernanceBlocked: true}, true, true, "frozen", "none", ""},
		{"staged verified", users.EntityComplete{IsActivated: 1, Email: user.Email, PendingEmail: "next@example.test", PendingEmailAt: &now}, true, true, "ready", "pending", "next@example.test"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := activationEligibility(c.user, now, c.enabled, c.mailReady)
			if got.State != c.state || got.TargetKind != c.kind || got.TargetEmail != c.target {
				t.Fatalf("got %+v", got)
			}
		})
	}
	old := now.Add(-users.PendingEmailWindow - time.Second)
	user.PendingEmail = "stale@example.test"
	user.PendingEmailAt = &old
	if got := activationEligibility(user, now, true, true); got.PendingEmail != "" || got.TargetEmail != user.Email {
		t.Fatalf("stale staging leaked: %+v", got)
	}
}

func TestActivationStatusQuotaUsesServerRelativeDelay(t *testing.T) {
	now := time.Date(2026, 10, 30, 23, 59, 0, 100000000, time.UTC)
	ready := ActivationStatus{State: "ready", RemainingToday: 3, DailyLimit: 3}
	got := statusWithQuota(ready, resendState{Count: 1, LastSentAt: now.Unix()}, now)
	if got.State != "cooldown" || got.RemainingToday != 2 || got.RetryAfterSeconds != 60 {
		t.Fatalf("fraction truncated: %+v", got)
	}
	got = statusWithQuota(ready, resendState{Count: 3, LastSentAt: now.Unix()}, now)
	if got.State != "daily_limit" || got.RemainingToday != 0 || got.RetryAfterSeconds != 60 {
		t.Fatalf("daily reset: %+v", got)
	}
	got = statusWithQuota(ready, resendState{Count: 2, LastSentAt: now.Add(-time.Minute).Unix()}, now)
	if got.State != "ready" || got.RemainingToday != 1 || got.RetryAfterSeconds != 0 {
		t.Fatalf("expired cooldown: %+v", got)
	}
}
