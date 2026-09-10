package api

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oauthservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
)

// ProviderLogin 开始OAuth登录/绑定流程（根据登录状态自动判断）
func ProviderLogin(c *gin.Context) {
	q := c.Request.URL.Query()
	q.Set("provider", c.Param("provider"))
	c.Request.URL.RawQuery = q.Encode()
	// Account binding retains its settings destination, even if a caller supplies
	// an OIDC continuation. Existing browser authentication is authoritative here.
	if component.GetLoginUser(c).UserId > 0 {
		q.Del("redirect")
		c.Request.URL.RawQuery = q.Encode()
	}
	// 开始 OAuth 流程
	oauthservice.BeginOAuthAuthHandler(c.Writer, c.Request)
}

// ProviderCallback 处理OAuth登录/绑定回调（根据登录状态自动判断）
func ProviderCallback(c *gin.Context) {
	q := c.Request.URL.Query()
	q.Set("provider", c.Param("provider"))
	c.Request.URL.RawQuery = q.Encode()

	// 完成 OAuth 流程
	gothUser, continuation, err := oauthservice.CompleteOAuthUserAuth(c.Writer, c.Request)
	if err != nil {
		slog.Error("OAuth callback failed", "error", err)
		forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthCallbackFailed)
		return
	}

	// 检查是否为绑定模式（用户已登录）
	currentUserInfo := component.GetLoginUser(c)
	currentUserId := currentUserInfo.UserId

	// A signed OIDC continuation was started as login, not account binding.
	if currentUserId > 0 && continuation == "" {
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
			if errors.Is(err, oauthservice.ErrOAuthNoLocalAccount) {
				// 纯新号（issue #531）：OAuth 回调不再建号，注册统一走
				// /api/register（allowedDomains 白名单在注册单点把关）。
				// 302 回注册页并携带提示参数。回跳目标优先级（PR #552 review P1）：
				// ① OIDC 续跳 continuation——内置 OIDC 桥接用户的注册完成页必须
				//   回到 /api/oauth/authorize/callback?id=…，移动端才能拿到授权码；
				//   该值已由 oidcResumeTarget 校验为站内桥接回调单参数形态；
				// ② 普通 Web 登录透传回调查询串 redirect（IsSafeRedirect 校验，
				//   不安全值静默丢弃，与登录页 props 同规则）。
				target := urlconfig.Register() + "?register=true&oauthNotice=1"
				if continuation != "" {
					target += "&redirect=" + url.QueryEscape(continuation)
				} else if redirect := c.Query("redirect"); forum.IsSafeRedirect(redirect) {
					target += "&redirect=" + url.QueryEscape(redirect)
				}
				slog.Info("OAuth callback without local account, redirecting to register", "provider", gothUser.Provider)
				c.Redirect(http.StatusFound, target)
				return
			}
			slog.Error("Process OAuth callback failed", "error", err)
			forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthProcessFailed)
			return
		}

		if user.IsActivated == users.ActivationPending {
			// 待激活 OAuth 用户同样发放会话：写权限在权限层由
			// CheckWritableAccount 拦截（permission.emailRequired），会话不授予
			// 写能力，用户借此可在会话过期后重新登录并继续激活恢复流程
			// （issue #427，与密码登录对 pending 用户的放行口径一致）。
			slog.Info("OAuth callback issued session for pending activation user",
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
		if continuation == "" {
			continuation = "/"
		}
		c.Redirect(http.StatusFound, continuation)
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
