package emailactivationservice

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/kvstore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
)

// ActivationStatus is private owner data. Reading it never reserves quota,
// generates a token, or enqueues mail. Relative delays avoid client clock skew.
type ActivationStatus struct {
	CurrentEmail      string `json:"currentEmail"`
	PendingEmail      string `json:"pendingEmail"`
	TargetEmail       string `json:"targetEmail"`
	TargetKind        string `json:"targetKind"`
	State             string `json:"state"`
	RemainingToday    int    `json:"remainingToday"`
	DailyLimit        int    `json:"dailyLimit"`
	RetryAfterSeconds int64  `json:"retryAfterSeconds"`
}

func GetStatus(user users.EntityComplete, now time.Time) (ActivationStatus, error) {
	status := activationEligibility(user, now, hotdataserve.GetSecuritySettingsConfigCache().EnableEmailVerification, mailservice.IsConfigured())
	key, _ := resendKey(user.Id, now)
	data, err := kvstore.GetBytes(key)
	if errors.Is(err, kvstore.ErrNotFound) {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	var state resendState
	if err := json.Unmarshal(data, &state); err != nil {
		return status, err
	}
	return statusWithQuota(status, state, now), nil
}

func activationEligibility(user users.EntityComplete, now time.Time, enabled, mailReady bool) ActivationStatus {
	status := ActivationStatus{CurrentEmail: user.Email, PendingEmail: user.FreshPendingEmail(now), TargetKind: "none", State: "ready", DailyLimit: resendDailyLimit, RemainingToday: resendDailyLimit}
	if status.PendingEmail != "" {
		status.TargetKind = "pending"
		status.TargetEmail = status.PendingEmail
	} else if user.IsActivated == users.ActivationPending && user.Email != "" {
		status.TargetKind = "current"
		status.TargetEmail = user.Email
	}
	switch {
	case user.IsFrozen == users.StatusFrozen || user.AnonymousGovernanceBlocked:
		status.State = "frozen"
	case !enabled:
		status.State = "disabled"
	case user.IsActivated != users.ActivationPending && status.PendingEmail == "":
		status.State = "verified"
	case status.TargetEmail == "":
		status.State = "missing_email"
	case !mailReady:
		status.State = "mail_unavailable"
	}
	return status
}

func statusWithQuota(status ActivationStatus, state resendState, now time.Time) ActivationStatus {
	status.RemainingToday = min(max(resendDailyLimit-state.Count, 0), resendDailyLimit)
	if status.State != "ready" {
		return status
	}
	if status.RemainingToday == 0 {
		status.State = "daily_limit"
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		status.RetryAfterSeconds = ceilSeconds(tomorrow.Sub(now))
	} else if delay := time.Unix(state.LastSentAt, 0).Add(resendInterval).Sub(now); state.LastSentAt > 0 && delay > 0 {
		status.State = "cooldown"
		status.RetryAfterSeconds = ceilSeconds(delay)
	}
	return status
}

func ceilSeconds(delay time.Duration) int64 { return max(int64((delay+time.Second-1)/time.Second), 0) }
