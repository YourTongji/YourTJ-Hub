package controllers

import (
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/resource"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	token := c.Query("token")
	if strings.TrimSpace(token) == "" {
		renderActivationPage(c, false, "activationMissingLink")
		return
	}

	// 解析激活令牌
	claims, err := tokenservice.ParseActivationToken(token)
	if err != nil {
		message := "activationInvalidLink"
		if errors.Is(err, jwt.ErrTokenExpired) {
			message = "activationExpired"
		}
		renderActivationPage(c, false, message)
		return
	}

	// 获取用户信息
	user, err := users.Get(claims.UserId)
	if err != nil {
		renderActivationPage(c, false, "activationInvalidLink")
		return
	}

	// 换绑确认：令牌目标命中新鲜暂存 → 原子切换。
	if user.FreshPendingEmail(time.Now()) == claims.Email {
		switched, err := users.CompletePendingEmailSwitch(user.Id, claims.Email, time.Now())
		if err != nil {
			renderActivationPage(c, false, "activationInvalidLink")
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
		renderActivationSuccess(c, &switched.User)
		return
	}

	// 普通激活：邮箱必须匹配当前 email。
	if user.Email != claims.Email {
		renderActivationPage(c, false, "activationInvalidLink")
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
		renderActivationPage(c, false, "activationInvalidLink")
		return
	}
	user.IsActivated = users.ActivationSuccess
	userservice.InvalidateUserInfoCache(user.Id)

	renderActivationSuccess(c, &user)
}

// ActivateAccountData 激活页面数据
type ActivateAccountData struct {
	SiteName    string
	RecoveryURL string
	SupportURL  string
	Title       string
	Message     string
	Success     bool
	Description string
	Account     string
	ContinueURL string
	LoginURL    string
}

// renderActivationPage 渲染账号激活结果页。messageKey 为 i18n 文案键。
func renderActivationSuccess(c *gin.Context, user *users.EntityComplete) {
	c.Set("activationAccount", user.Username)
	renderActivationPage(c, true, "activationSuccess")
}

func renderActivationPage(c *gin.Context, success bool, messageKey string) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	lang := component.RequestLang(c)
	tr := i18n.Func(lang)

	title := tr("activationTitleFail")
	description := tr("activationRecoveryDescription")
	if success {
		title = tr("activationTitleSuccess")
		description = tr("activationDescSuccess")
	}

	site := hotdataserve.GetSiteSettingsConfigCache()
	name := strings.TrimSpace(site.SiteName)
	if name == "" {
		name = "yourtj"
	}
	supportURL := ""
	email := strings.TrimSpace(site.SiteEmail)
	// The shipped example address is not a configured support destination.
	if !strings.EqualFold(email, strings.TrimSpace(defaultconfig.GetDefaultSiteSettingsConfig().SiteEmail)) {
		if address, err := mail.ParseAddress(email); err == nil && !strings.ContainsAny(address.Address, "\r\n") {
			supportURL = "mailto:" + url.PathEscape(address.Address)
		}
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
			SiteName:    name,
			RecoveryURL: "/settings?tab=profile",
			SupportURL:  supportURL,
			Title:       title,
			Message:     tr(messageKey),
			Success:     success,
			Description: description,
			Account:     c.GetString("activationAccount"),
			ContinueURL: activationContinuation(c.Query("redirect")),
			LoginURL:    activationLoginURL(c.Query("redirect")),
		},
		T:    tr,
		Lang: lang,
	}); err != nil {
		slog.Error("render activation template failed", "err", err)
	}
}
