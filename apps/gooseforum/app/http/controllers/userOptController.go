package controllers

import (
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/resource"
	"github.com/gin-gonic/gin"
)

var activationTemplate = sync.OnceValues(func() (*template.Template, error) {
	return template.ParseFS(resource.GetTemplateFS(), "templates/view/activate.gohtml")
})

// ActivateAccount 激活处理函数。
// 两阶段换绑（issue #678）：激活令牌绑定签发时的目标邮箱。若目标命中仍在
// 窗口内的换绑暂存，则原子完成第二阶段切换（email ← pending、清暂存、置
// 已激活、刷新 24h 找回冷静期起点），并向旧邮箱发送最终变更通知；
// 否则按普通账号激活处理（目标必须是当前 email，残留暂存一并清除——
// 用户选择了验证当前邮箱而不是继续换绑）。
func ActivateAccount(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		renderActivationPage(c, false, "activationInvalidLink")
		return
	}

	// 解析激活令牌
	claims, err := tokenservice.ParseActivationToken(token)
	if err != nil {
		renderActivationPage(c, false, "activationExpired")
		return
	}

	// 获取用户信息
	user, err := users.Get(claims.UserId)
	if err != nil {
		renderActivationPage(c, false, "activationUserNotFound")
		return
	}

	// 换绑确认：令牌目标命中新鲜暂存 → 原子切换。
	if user.FreshPendingEmail(time.Now()) == claims.Email {
		switched, err := users.CompletePendingEmailSwitch(user.Id, claims.Email, time.Now())
		if err != nil {
			renderActivationPage(c, false, "activationLinkInvalid")
			return
		}
		userservice.RefreshUserCaches(&switched.User)
		// 旧邮箱：最终变更通知（受害者获知换绑完成的主要途径，失败需 Error 暴露）。
		if switched.OldEmail != "" {
			if err := mailservice.AddToQueue(mailservice.EmailTask{
				To:       switched.OldEmail,
				Username: switched.User.Username,
				NewEmail: switched.User.Email,
				Type:     "email_changed",
				Locale:   switched.User.Locale,
			}); err != nil {
				slog.Error("换绑完成通知入队失败", "userId", switched.User.Id, "oldEmail", switched.OldEmail, "error", err)
			}
		}
		renderActivationPage(c, true, "activationSuccess")
		return
	}

	// 普通激活：邮箱必须匹配当前 email。
	if user.Email != claims.Email {
		renderActivationPage(c, false, "activationLinkInvalid")
		return
	}

	// 激活当前邮箱并清除残留的换绑暂存（用户放弃了换绑，选择验证当前邮箱）。
	// CAS 条件更新（issue #678 review P2）：约束读取时的 email 与 pending_email，
	// 若并发的新邮箱确认切换已先完成，本次不命中并按链接无效渲染——避免
	// 全行 Save 把已切换的 email 回写成旧值。
	stagedBefore := user.PendingEmail
	applied, err := users.ActivateCurrentEmail(user.Id, user.Email, stagedBefore, time.Now())
	if err != nil {
		renderActivationPage(c, false, "activationFailed")
		return
	}
	if !applied {
		renderActivationPage(c, false, "activationLinkInvalid")
		return
	}
	user.IsActivated = users.ActivationSuccess
	userservice.InvalidateUserInfoCache(user.Id)

	renderActivationPage(c, true, "activationSuccess")
}

// ActivateAccountData 激活页面数据
type ActivateAccountData struct {
	Title       string
	Message     string
	Success     bool
	Description string
}

// renderActivationPage 渲染账号激活结果页。messageKey 为 i18n 文案键。
func renderActivationPage(c *gin.Context, success bool, messageKey string) {
	lang := component.RequestLang(c)
	tr := i18n.Func(lang)

	title := tr("activationTitleFail")
	description := tr("activationDescFail")
	if success {
		title = tr("activationTitleSuccess")
		description = tr("activationDescSuccess")
	}

	tmpl, err := activationTemplate()
	if err != nil {
		slog.Error("parse activation template failed", "err", err)
		c.String(http.StatusInternalServerError, "activation page unavailable")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.Execute(c.Writer, struct {
		Data ActivateAccountData
		T    func(string, ...any) string
		Lang string
	}{
		Data: ActivateAccountData{
			Title:       title,
			Message:     tr(messageKey),
			Success:     success,
			Description: description,
		},
		T:    tr,
		Lang: lang,
	}); err != nil {
		slog.Error("render activation template failed", "err", err)
	}
}
