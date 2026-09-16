package emailactivationservice

import (
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
)

func SendActivationEmail(userEntity *users.EntityComplete) error {
	token, err := tokenservice.GenerateActivationTokenByUser(*userEntity)
	if err != nil {
		slog.Debug("生成激活邮件 Token 失败", "userId", userEntity.Id, "email", userEntity.Email, "err", err)
		return err
	}

	err = mailservice.AddToQueue(mailservice.EmailTask{
		To:       userEntity.Email,
		Username: userEntity.Username,
		Token:    token,
		Type:     "activation",
		Locale:   userEntity.Locale,
	})
	if err != nil {
		slog.Debug("激活邮件任务入队失败", "userId", userEntity.Id, "email", userEntity.Email, "err", err)
		return err
	}
	slog.Debug("激活邮件任务入队成功", "userId", userEntity.Id, "email", userEntity.Email)
	return nil
}

// SendPendingEmailActivation 发送两阶段换绑第二阶段的确认邮件（issue #678）：
// 令牌绑定 pendingEmail（而非当前 email），ActivateAccount 据此区分「激活当前
// 邮箱」与「确认换绑」两条路径，确认链接只在 pending 与 claims 一致时切换。
func SendPendingEmailActivation(userEntity *users.EntityComplete, pendingEmail string) error {
	token, err := tokenservice.GenerateActivationToken(userEntity.Id, pendingEmail)
	if err != nil {
		slog.Debug("生成换绑确认邮件 Token 失败", "userId", userEntity.Id, "pendingEmail", pendingEmail, "err", err)
		return err
	}

	err = mailservice.AddToQueue(mailservice.EmailTask{
		To:       pendingEmail,
		Username: userEntity.Username,
		Token:    token,
		Type:     "activation",
		Locale:   userEntity.Locale,
	})
	if err != nil {
		slog.Debug("换绑确认邮件任务入队失败", "userId", userEntity.Id, "pendingEmail", pendingEmail, "err", err)
		return err
	}
	slog.Debug("换绑确认邮件任务入队成功", "userId", userEntity.Id, "pendingEmail", pendingEmail)
	return nil
}
