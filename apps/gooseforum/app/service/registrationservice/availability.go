package registrationservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
	"log/slog"
)

// PasswordAvailability is public, advisory state for the ordinary email form.
// It exposes no counts or account lookup. Creation must recheck its own gates
// and the existing transactional signup quota; reading never reserves a slot.
func PasswordAvailability(config pageConfig.SecurityAndRegistration) string {
	if !config.EnableSignup {
		return "disabled"
	}
	if config.EnableEmailVerification && !mailservice.IsConfigured() {
		return "mail_unavailable"
	}
	if config.MaxDailySignups == 0 {
		return "daily_limit"
	}
	if config.MaxDailySignups > 0 {
		count, err := users.CountCreatedTodayWithError()
		if err != nil {
			slog.Error("read signup availability failed", "err", err)
			return "unavailable"
		}
		if count >= int64(config.MaxDailySignups) {
			return "daily_limit"
		}
	}
	return "open"
}
