package emailactivationservice

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/kvstore"
	"github.com/google/uuid"
)

const (
	resendDailyLimit = 3
	resendInterval   = time.Minute
)

var (
	ErrDisabled        = errors.New("email activation resend disabled")
	ErrAlreadyVerified = errors.New("email activation already verified")
	ErrCooldown        = errors.New("email activation resend cooldown")
	ErrDailyLimit      = errors.New("email activation resend daily limit")
)

type ResendResult struct {
	RemainingToday    int
	RetryAfterSeconds int64
	DailyLimit        int
}

type resendState struct {
	Count      int   `json:"count"`
	LastSentAt int64 `json:"lastSentAt"`
	// Legacy records have only Count/LastSentAt. Keep their charge and cooldown
	// as a baseline; the bounded entries identify only sends we can compensate.
	BaselineLastSentAt int64          `json:"baselineLastSentAt,omitempty"`
	Reservations       []reservedSend `json:"reservations,omitempty"`
}

type reservedSend struct {
	ID string `json:"id"`
	At int64  `json:"at"`
}

type quotaReservation struct {
	key string
	ttl time.Duration
	id  string
}

func Resend(userEntity users.EntityComplete) (ResendResult, error) {
	result := ResendResult{DailyLimit: resendDailyLimit}

	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	if !securityConfig.EnableEmailVerification {
		return result, ErrDisabled
	}

	now := time.Now()
	// 两阶段换绑（issue #678）：存在窗口内的换绑暂存时，重发对象是暂存邮箱的
	// 确认邮件（已激活账号也会走到这里；无暂存且已激活仍返回已验证错误）。
	pending := userEntity.FreshPendingEmail(now)
	if pending == "" && userEntity.IsActivated != users.ActivationPending {
		return result, ErrAlreadyVerified
	}

	result, reservation, err := reserveQuota(userEntity.Id, now)
	if err != nil {
		return result, err
	}

	if pending != "" {
		err = SendPendingEmailActivation(&userEntity, pending)
	} else {
		err = SendActivationEmail(&userEntity)
	}
	if err != nil {
		slog.Info("验证邮件重发失败", "userId", userEntity.Id, "error", err)
		// Only a synchronous token/queue failure is compensated. Once enqueued,
		// neither HTTP cancellation nor a later SMTP failure refunds the charge.
		if released, releaseErr := releaseQuota(reservation); releaseErr != nil {
			slog.Error("返还验证邮件重发额度失败", "userId", userEntity.Id, "error", releaseErr)
			err = errors.Join(err, releaseErr)
		} else {
			result = released
		}
		return result, err
	}

	return result, nil
}

func reserveQuota(userID uint64, now time.Time) (ResendResult, quotaReservation, error) {
	result := ResendResult{DailyLimit: resendDailyLimit}
	key, ttl := resendKey(userID, now)
	reservation := quotaReservation{key: key, ttl: ttl, id: uuid.NewString()}
	err := kvstore.UpdateBytes(key, ttl, func(current []byte, exists bool) (kvstore.UpdateAction, []byte, error) {
		result = ResendResult{DailyLimit: resendDailyLimit}
		state := resendState{}
		if exists && len(current) > 0 {
			if err := json.Unmarshal(current, &state); err != nil {
				state = resendState{}
			}
		}

		if state.LastSentAt > 0 {
			nextAllowedAt := time.Unix(state.LastSentAt, 0).Add(resendInterval)
			if now.Before(nextAllowedAt) {
				result.RetryAfterSeconds = max(int64(nextAllowedAt.Sub(now).Seconds()), 1)
				return kvstore.UpdateKeep, nil, ErrCooldown
			}
		}
		if state.Count >= resendDailyLimit {
			return kvstore.UpdateKeep, nil, ErrDailyLimit
		}

		if len(state.Reservations) == 0 {
			state.BaselineLastSentAt = state.LastSentAt
		}
		state.Count++
		state.LastSentAt = now.Unix()
		state.Reservations = append(state.Reservations, reservedSend{ID: reservation.id, At: now.Unix()})
		data, err := json.Marshal(state)
		if err == nil {
			result.RemainingToday = max(resendDailyLimit-state.Count, 0)
		}
		return kvstore.UpdateSet, data, err
	})
	return result, reservation, err
}

// releaseQuota removes exactly the failed send, even if a slower request
// finishes after a later send. Successful reservations stay charged until the
// daily key expires (at most three entries); no second commit is needed after
// enqueueing. Retaining the original key also isolates midnight compensation.
func releaseQuota(reservation quotaReservation) (ResendResult, error) {
	result := ResendResult{DailyLimit: resendDailyLimit}
	err := kvstore.UpdateBytes(reservation.key, reservation.ttl, func(current []byte, exists bool) (kvstore.UpdateAction, []byte, error) {
		result = ResendResult{DailyLimit: resendDailyLimit}
		if !exists {
			return kvstore.UpdateKeep, nil, nil
		}
		var state resendState
		if err := json.Unmarshal(current, &state); err != nil {
			return kvstore.UpdateKeep, nil, err
		}
		result.RemainingToday = max(resendDailyLimit-state.Count, 0)
		for i, entry := range state.Reservations {
			if entry.ID != reservation.id {
				continue
			}
			state.Reservations = append(state.Reservations[:i], state.Reservations[i+1:]...)
			state.Count--
			state.LastSentAt = state.BaselineLastSentAt
			for _, remaining := range state.Reservations {
				state.LastSentAt = max(state.LastSentAt, remaining.At)
			}
			result.RemainingToday = max(resendDailyLimit-state.Count, 0)
			data, err := json.Marshal(state)
			return kvstore.UpdateSet, data, err
		}
		// Repeated or stale compensation cannot release another request's slot.
		return kvstore.UpdateKeep, nil, nil
	})
	return result, err
}

func resendKey(userID uint64, now time.Time) (string, time.Duration) {
	day := now.Format("20060102")
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	ttl := max(time.Until(tomorrow)+time.Hour, time.Hour)
	return "activation-email:resend:" + strconv.FormatUint(userID, 10) + ":" + day, ttl
}
