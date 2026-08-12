package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oauthservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

// ProviderLogin 开始OAuth登录/绑定流程（根据登录状态自动判断）
func ProviderLogin(c *gin.Context) {
	q := c.Request.URL.Query()
	q.Add("provider", c.Param("provider"))
	c.Request.URL.RawQuery = q.Encode()
	// 开始 OAuth 流程
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

// ProviderCallback 处理OAuth登录/绑定回调（根据登录状态自动判断）
func ProviderCallback(c *gin.Context) {
	q := c.Request.URL.Query()
	q.Add("provider", c.Param("provider"))
	c.Request.URL.RawQuery = q.Encode()

	// 完成 OAuth 流程
	gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		slog.Error("OAuth callback failed", "error", err)
		forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthCallbackFailed)
		return
	}

	// 检查是否为绑定模式（用户已登录）
	currentUserInfo := component.GetLoginUser(c)
	currentUserId := currentUserInfo.UserId

	if currentUserId > 0 {
		if user, ok := userservice.GetUserInfo(currentUserId); !ok || user.IsFrozen == users.StatusFrozen || user.ActorType == users.ActorTypeBot {
			forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessagePermissionUserFrozen)
			return
		}

		// 绑定模式：处理OAuth绑定
		err = oauthservice.ProcessOAuthBind(currentUserId, gothUser)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/settings?tab=binding")
			return
		}
		// 绑定成功，重定向到账户设置页面
		c.Redirect(http.StatusTemporaryRedirect, "/settings?tab=binding")
	} else {
		// 登录模式：处理OAuth登录
		user, err := oauthservice.ProcessOAuthCallback(gothUser)
		if err != nil {
			// 冻结账号：service 层返回 ErrAccountFrozen，禁止通过 OAuth 重新获取会话，
			// 这里渲染 403 冻结错误页，与 OIDC exchange 的冻结语义保持一致。
			// 冻结拒绝是用户可预期触发的路径，用 Warn 记录，避免日志噪音。
			if errors.Is(err, oauthservice.ErrAccountFrozen) {
				slog.Warn("OAuth callback rejected frozen account", "error", err)
				forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessageOAuthAccountFrozen)
				return
			}
			slog.Error("Process OAuth callback failed", "error", err)
			forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthProcessFailed)
			return
		}

		if user.IsActivated == users.ActivationPending {
			// issue #155：OAuth 用户若处于待激活状态（verified 邮箱未命中信任域名且
			// 全局 EnableEmailVerification 开启），不得在回调中强制激活。
			// 保持 ActivationPending，用户需通过激活邮件完成验证，
			// 与密码注册流程的激活语义一致。
			slog.Info("OAuth callback user pending activation",
				"userId", user.Id, "provider", gothUser.Provider)
		}

		// 生成JWT token（会话凭证，写会话记录）
		token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
		if err != nil {
			slog.Error("Generate JWT token failed", "error", err)
			forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthTokenFailed)
			return
		}
		if err = sessionservice.Create(user.Id, jti, c.Request.UserAgent(), c.ClientIP()); err != nil {
			slog.Error("Create OAuth session failed", "userId", user.Id, "error", err)
			forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthProcessFailed)
			return
		}

		jwtopt.TokenSetting(c, token)
		c.Redirect(http.StatusFound, "/")
	}
}

// UnbindOAuth 解绑OAuth账户
func UnbindOAuth(req component.BetterRequest[component.Null]) component.Response {
	// 检查用户是否已登录
	userID := req.UserId

	provider := req.GinContext.Param("provider")

	// 解绑OAuth账户
	err := oauthservice.UnbindOAuth(userID, provider)
	if err != nil {
		return component.FailResponseCode(
			component.MessageOAuthUnbindFailed,

			component.MessageParams{"error": err.Error(), "provider": provider})

	}
	return component.SuccessResponseCode("解绑成功", component.MessageOAuthUnbindSuccess, component.MessageParams{"provider": provider})
}

// GetOAuthBindings 获取用户的OAuth绑定状态
func GetOAuthBindings(req component.BetterRequest[component.Null]) component.Response {
	// 检查用户是否已登录
	userID := req.UserId

	// 获取用户的 OAuth 绑定
	bindings := oauthservice.GetUserOAuthBindings(userID)

	// 构建响应数据
	result := make(map[string]any)
	for provider, oauth := range bindings {
		result[provider] = map[string]any{
			"bound":     true,
			"provider":  oauth.Provider,
			"createdAt": oauth.CreatedAt,
			"updatedAt": oauth.UpdatedAt,
		}
	}

	// 添加未绑定的提供商
	allProviders := []string{"github", "google"}
	for _, provider := range allProviders {
		if _, exists := result[provider]; !exists {
			result[provider] = map[string]any{
				"bound": false,
			}
		}
	}
	return component.SuccessResponse(result)

}
