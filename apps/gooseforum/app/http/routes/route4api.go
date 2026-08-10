package routes

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-contrib/gzip"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/setting"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/controllers/forum"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/http/controllers"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	"github.com/leancodebox/GooseForum/app/service/oidcservice"
	"github.com/leancodebox/GooseForum/app/service/permission"
	"github.com/leancodebox/GooseForum/resource"
)

func gzipEnabled() bool {
	return preferences.GetBool("server.gzip", true)
}

func assertRouter(ginApp *gin.Engine) {
	assetsFs, _ := resource.GetAssetsFS()
	staticFS, _ := resource.GetStaticFS()
	staticRoute := ginApp.Group("/")
	if gzipEnabled() {
		staticRoute.Use(middleware.CacheMiddleware)
		staticRoute.Use(gzip.Gzip(gzip.DefaultCompression))
		slog.Info("static assets gzip enabled", "cache", true)
	} else {
		slog.Info("static assets gzip disabled", "cache", false)
	}
	// dev 模式：/assets/* 反向代理到 Vite 开发服务器（同源相对路径，本机与局域网均可访问）
	if devServer := viteDevServerURL(); devServer != "" {
		target, err := url.Parse(devServer)
		if err != nil {
			slog.Error("vite dev server url parse", "err", err)
		} else {
			proxy := httputil.NewSingleHostReverseProxy(target)
			staticRoute.Any("assets/*path", gin.WrapH(proxy))
			slog.Info("assets proxied to vite dev server", "target", devServer)
		}
	} else {
		staticRoute.StaticFS("assets", http.FS(assetsFs))
	}
	staticRoute.
		Use(middleware.BrowserCache).
		StaticFS("static", http.FS(staticFS))
}

func viteDevServerURL() string {
	// 生产环境无条件禁用 Vite 代理：即使配置了 resource.devServer，
	// 也不能把 /assets/* 代理到开发服务器（避免生产事故）。
	if setting.IsProduction() {
		return ""
	}
	return preferences.GetString("resource.devServer", "http://localhost:3010")
}

func viewRoute(ginApp *gin.Engine) {
	ginApp.GET("/reload", func(c *gin.Context) {
		if setting.IsProduction() {
			c.String(http.StatusNotFound, "404")
			return
		}
		forum.ReloadTemplates()
		c.String(200, "模板已刷新")
	})

	viewRouteApp := ginApp.Group("")
	viewRouteApp.Use(middleware.JWTAuth)
	if gzipEnabled() {
		viewRouteApp.Use(gzip.Gzip(gzip.DefaultCompression))
		slog.Info("view gzip enabled")
	} else {
		slog.Info("view gzip disabled")
	}

	viewRouteApp.GET("/", forum.Home)
	viewRouteApp.GET("/p/post/:id", forum.TopicDetail)
	viewRouteApp.GET("/p/post/:id/:postNo", forum.TopicDetail)
	viewRouteApp.GET("/u/:userId", forum.UserProfile)
	viewRouteApp.GET("/u/:userId/:section", forum.UserProfile)
	viewRouteApp.GET("/u/:userId/:section/:subsection", forum.UserProfile)
	viewRouteApp.GET("/c/:slug/:id", forum.Category)
	viewRouteApp.GET("/c/:slug/:id/l/:sort", forum.Category)
	viewRouteApp.GET("/links", forum.Links)
	viewRouteApp.GET("/sponsors", forum.Sponsors)
	viewRouteApp.GET("/messages", middleware.CheckLogin, forum.Messages)
	viewRouteApp.GET("/drafts", middleware.CheckLogin, forum.Drafts)
	viewRouteApp.GET("/moderation", middleware.CheckLogin, forum.Moderation)
	viewRouteApp.GET("/settings", middleware.CheckLogin, forum.Settings)
	viewRouteApp.GET("/theme-preview", middleware.CheckLogin, middleware.CheckAnyPermissionOrNotFound, forum.ThemePreview)
	viewRouteApp.GET("/notifications", middleware.CheckLogin, forum.Notifications)
	viewRouteApp.GET("/publish", middleware.CheckLogin, forum.Publish)
	viewRouteApp.GET("/search", forum.Search)
	viewRouteApp.GET("/admin", middleware.CheckLogin, middleware.CheckAnyPermissionOrNotFound, forum.Manage)
	viewRouteApp.GET("/admin/*path", middleware.CheckLogin, middleware.CheckAnyPermissionOrNotFound, forum.Manage)
	viewRouteApp.GET("/login", forum.Login)
	viewRouteApp.GET("/reset-password", forum.ResetPassword)
	viewRouteApp.GET("/terms", forum.Terms)

	viewRouteApp.GET("/activate", controllers.ActivateAccount)

	ginApp.GET("/site-theme.css", forum.SiteThemeCSS)
}

func siteInfoRoute(ginApp *gin.Engine) {
	ginApp.GET("/robots.txt", controllers.RenderRobotsTxt)
	ginApp.GET("/sitemap.xml", controllers.RenderSitemapXml)
	ginApp.GET("/rss.xml", controllers.RenderRss)
	// LLMS 公开入口是派生文本投影：full 冷缓存全量重建、不同 {id}.md 会打穿 topic 级缓存，
	// 需独立限流配额（index 宽松、full 严格、topic 中等），避免被脚本高频打满。
	ginApp.GET("/llms.txt", middleware.RateLimit(middleware.RateLimitLLMSIndex), controllers.RenderLLMSIndex)
	ginApp.GET("/llms-full.txt", middleware.RateLimit(middleware.RateLimitLLMSFull), controllers.RenderLLMSFull)
	ginApp.GET("/p/posts/:document", middleware.RateLimit(middleware.RateLimitLLMSTopic), controllers.RenderLLMSTopic)
}

func apiRoute(ginApp *gin.Engine) {
	baseApi := ginApp.Group("api")

	baseApi.POST("login", middleware.RateLimit(middleware.RateLimitLogin), api.Login)
	baseApi.GET("login-public-key", api.LoginPublicKey)
	baseApi.POST("register", middleware.RateLimit(middleware.RateLimitRegister), api.Register)
	baseApi.POST("logout", api.Logout)

	baseApi.GET("get-captcha", UpQueryReq(api.GetCaptcha))
	baseApi.GET("user-card", UpQueryReq(api.GetUserCard))
	baseApi.POST("forgot-password", middleware.RateLimit(middleware.RateLimitForgotPassword), UpButterReq(api.ForgotPassword))
	baseApi.POST("reset-password", UpButterReq(api.ResetPassword))
	baseApi.GET("auth/:provider", api.ProviderLogin)
	baseApi.GET("auth/:provider/callback", middleware.JWTAuth, api.ProviderCallback)

	// 内建 OIDC Provider（/api/oauth）。逐个静态挂载已实现端点，避免
	// oauth/*path catch-all 与论坛自身的 oauth/bindings 路由发生 Gin 冲突。
	// 未启用或配置错误时不注册路由，但错误必须记录（生产误配 fail closed
	// 时便于排查）。authorize 与 token 使用独立配额，且在进入 Provider 前限流。
	if oidcHandler, err := oidcservice.Router(); err != nil {
		if !errors.Is(err, oidcservice.ErrOIDCDisabled) {
			slog.Error("OIDC provider router init failed", "error", err)
		}
	} else if oidcHandler != nil {
		wrapped := gin.WrapH(oidcHandler)
		baseApi.GET("oauth/.well-known/openid-configuration", wrapped)
		baseApi.GET("oauth/authorize", middleware.RateLimit(middleware.RateLimitOIDCAuthorize), wrapped)
		baseApi.GET("oauth/authorize/callback", wrapped)
		baseApi.GET("oauth/token", middleware.RateLimit(middleware.RateLimitOIDCToken), wrapped)
		baseApi.POST("oauth/token", middleware.RateLimit(middleware.RateLimitOIDCToken), wrapped)
		baseApi.GET("oauth/userinfo", wrapped)
		baseApi.POST("oauth/userinfo", wrapped)
		baseApi.GET("oauth/keys", wrapped)
	}

	baseApi.POST("auth/totp/verify", middleware.TOTPChallengeAuth, api.TotpVerify)
	baseApi.POST("auth/oidc/exchange", middleware.RateLimit(middleware.RateLimitLogin), api.OidcExchange)

	loginApi := ginApp.Group("api").Use(middleware.JWTAuthCheck)
	loginApi.POST("set-user-info", middleware.CheckWritableAccount, UpButterReq(api.EditUserInfo))
	loginApi.POST("set-user-profile-cover", middleware.CheckWritableAccount, UpButterReq(api.EditUserProfileCover))
	loginApi.POST("set-user-email", middleware.CheckWritableAccount, UpButterReq(api.EditUserEmail))
	loginApi.POST("resend-activation-email", middleware.CheckWritableAccount, UpButterReq(api.ResendActivationEmail))
	loginApi.POST("set-user-name", middleware.CheckWritableAccount, UpButterReq(api.EditUsername))
	loginApi.POST("set-preset-avatar", middleware.CheckWritableAccount, UpButterReq(api.SetPresetAvatar))
	loginApi.POST("wear-badge", middleware.CheckWritableAccount, UpButterReq(api.WearBadge))
	loginApi.POST("upload-avatar", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.UploadAvatar)
	loginApi.POST("change-password", middleware.CheckWritableAccount, UpButterReq(api.ChangePassword))
	loginApi.POST("auth/:provider/unbind", middleware.CheckWritableAccount, UpButterReq(api.UnbindOAuth))
	loginApi.GET("oauth/bindings", UpButterReq(api.GetOAuthBindings))
	loginApi.GET("user/sessions", UpButterReq(api.ListSessions))
	loginApi.POST("user/sessions/revoke", UpButterReq(api.RevokeSession))
	loginApi.POST("user/sessions/revoke-all", UpButterReq(api.RevokeAllSessions))
	loginApi.POST("user/totp/setup", UpButterReq(api.TotpSetup))
	loginApi.POST("user/totp/enable", UpButterReq(api.TotpEnable))
	loginApi.POST("user/totp/disable", UpButterReq(api.TotpDisable))
	loginApi.GET("user/totp/status", UpButterReq(api.TotpStatus))

	forumApi := baseApi.Group("forum")
	forumApi.GET("get-site-statistics", ginUpNP(api.GetSiteStatistics))
	forumApi.GET("search", middleware.JWTAuth, UpQueryReq(forum.SearchJSON))
	forumApi.GET("posts/window", middleware.JWTAuth, middleware.NoUpdateUserActivity, UpQueryReq(forum.PostWindow))

	forumLoginApi := forumApi.Use(middleware.JWTAuthCheck)
	forumLoginApi.GET("unread-status", middleware.NoUpdateUserActivity, UpButterReq(api.GetUnreadStatus))
	forumLoginApi.GET("notifications", middleware.NoUpdateUserActivity, UpQueryReq(api.NotificationList))
	forumLoginApi.POST("notification/mark-read", middleware.CheckWritableAccount, UpButterReq(api.MarkAsRead))
	forumLoginApi.POST("notification/mark-all-read", middleware.CheckWritableAccount, UpButterReq(api.MarkAllAsRead))
	forumLoginApi.POST("topics/write", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitTopicWrite), UpButterReq(api.WriteTopic))
	forumLoginApi.POST("topics/status", middleware.CheckWritableAccount, UpButterReq(api.UpdateTopicStatus))
	forumLoginApi.POST("topics/delete", middleware.CheckWritableAccount, UpButterReq(api.DeleteTopicByUser))
	forumLoginApi.GET("user/deleted-content", middleware.NoUpdateUserActivity, UpQueryReq(api.DeletedContentList))
	forumLoginApi.POST("user/content-restore", middleware.CheckWritableAccount, UpButterReq(api.RestoreContent))
	forumLoginApi.POST("user/content-purge", middleware.CheckWritableAccount, UpButterReq(api.PurgeContent))
	forumLoginApi.POST("user/content-privacy-erase", middleware.CheckWritableAccount, UpButterReq(api.PrivacyErase))
	forumLoginApi.POST("user/content-event", middleware.CheckWritableAccount, UpButterReq(api.ReportContentEvent))
	forumLoginApi.POST("posts/create", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostCreate), UpButterReq(api.CreatePost))
	forumLoginApi.POST("posts/update", middleware.CheckWritableAccount, UpButterReq(api.UpdatePost))
	forumLoginApi.POST("posts/delete", middleware.CheckWritableAccount, UpButterReq(api.DeletePost))
	forumLoginApi.POST("posts/like", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.LikePost))
	forumLoginApi.POST("posts/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BookmarkPost))
	forumLoginApi.POST("topics/like", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.LikeTopic))
	forumLoginApi.POST("topics/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BookmarkTopic))
	forumLoginApi.POST("topics/watch", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.WatchTopic))
	forumLoginApi.POST("follow-user", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.FollowUser))
	forumLoginApi.POST("report", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(forum.CreateReport))
	forumLoginApi.POST("moderation/topic-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationTopicStatus))
	forumLoginApi.POST("moderation/post-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationPostStatus))
	forumLoginApi.POST("moderation/reports", middleware.NoUpdateUserActivity, UpButterReq(forum.ModerationReportList))
	forumLoginApi.POST("moderation/report-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationReportStatus))
	forumLoginApi.POST("moderation/logs", middleware.NoUpdateUserActivity, UpButterReq(forum.ModerationLogList))
	forumLoginApi.POST("moderation/view-deleted-content", middleware.CheckWritableAccount, UpButterReq(forum.ViewDeletedContent))

	chatApi := forumApi.Group("chat", middleware.JWTAuthCheck)

	// Agent public API: opaque bearer-token authentication only. Writes reuse
	// the human topic/post rate limits keyed by IP and bot userId.
	agentApi := baseApi.Group("v1/agent", middleware.AgentAuth)
	agentApi.GET("me", UpButterReq(api.AgentMe))
	agentApi.GET("topics", UpQueryReq(api.AgentTopicList))
	agentApi.POST("topics", middleware.RateLimit(middleware.RateLimitTopicWrite), UpJsonReq(api.AgentWriteTopic))
	agentApi.GET("topics/:topicId/posts", UpUriQueryReq(api.AgentPostList))
	agentApi.POST("topics/:topicId/posts", middleware.RateLimit(middleware.RateLimitPostCreate), UpUriJsonReq(api.AgentCreatePost))
	agentApi.GET("search", UpQueryReq(forum.SearchJSON))
	chatApi.POST("send", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitMessageSend), UpButterReq(api.SendMessage))
	chatApi.POST("messages", UpButterReq(api.GetMessages))
	chatApi.POST("mark-read", middleware.CheckWritableAccount, UpButterReq(api.MarkChatRead))

	adminApi := baseApi.Group("admin", middleware.JWTAuthCheck, middleware.CheckWritableAccount)

	adminApi.POST("traffic-overview", middleware.CheckPermission(permission.Admin), UpButterReq(api.GetTrafficOverview))

	adminApi.
		Group("", middleware.CheckPermission(permission.UserManager)).
		POST("user-list", UpButterReq(api.UserList)).
		POST("user-edit", UpButterReq(api.EditUser)).
		POST("user-badge-options", UpButterReq(api.UserBadgeOptions)).
		POST("save-user-badges", UpButterReq(api.SaveUserBadges)).
		GET("get-all-role-item", UpButterReq(api.GetAllRoleItem))

	adminApi.Group("", middleware.CheckPermission(permission.TopicsManager)).
		POST("topics/list", UpButterReq(api.TopicsList)).
		POST("topics/source", UpButterReq(api.TopicSource)).
		POST("topics/edit", UpButterReq(api.EditTopic)).
		POST("topics/delete", UpButterReq(api.DeleteTopic)).
		POST("posts/delete", UpButterReq(api.DeletePostAsModerator)).
		POST("topics/pin-edit", UpButterReq(api.EditTopicPin)).
		POST("topics/categories-edit", UpButterReq(api.EditTopicCategories)).
		POST("category-list", UpButterReq(api.GetCategoryList)).
		POST("category-save", UpButterReq(api.SaveCategory)).
		POST("category-delete", UpButterReq(api.DeleteCategory)).
		POST("global-moderator-list", UpButterReq(api.GetGlobalModeratorList)).
		POST("global-moderator-add", UpButterReq(api.AddGlobalModerator)).
		POST("global-moderator-delete", UpButterReq(api.DeleteGlobalModerator)).
		POST("category-moderator-add", UpButterReq(api.AddCategoryModerator)).
		POST("category-moderator-delete", UpButterReq(api.DeleteCategoryModerator))

	adminApi.Group("", middleware.CheckPermission(permission.RoleManager)).
		POST("get-permission-list", UpButterReq(api.GetPermissionList)).
		POST("role-list", UpButterReq(api.RoleList)).
		POST("role-save", UpButterReq(api.RoleSave)).
		POST("role-delete", UpButterReq(api.RoleDel))

	adminApi.Group("", middleware.CheckPermission(permission.Admin)).
		POST("opt-record-page", UpButterReq(api.OptRecordPage)).
		POST("agent-list", UpButterReq(api.AgentList)).
		POST("agent-create", UpButterReq(api.AgentCreate)).
		POST("agent-update", UpButterReq(api.AgentUpdate)).
		POST("agent-rotate-token", UpButterReq(api.AgentRotateToken)).
		POST("agent-disable", UpButterReq(api.AgentDisable))

	adminApi.Group("", middleware.CheckPermission(permission.PageManager)).
		GET("friend-links", UpButterReq(api.GetFriendLinks)).
		POST("save-friend-links", UpButterReq(api.SaveFriendLinks)).
		GET("sponsors", UpButterReq(api.GetSponsors)).
		POST("save-sponsors", UpButterReq(api.SaveSponsors)).
		GET("announcement", UpButterReq(api.GetAnnouncement)).
		POST("save-announcement", UpButterReq(api.SaveAnnouncement))

	adminApi.Group("", middleware.CheckPermission(permission.SiteManager)).
		GET("server-version", UpButterReq(api.ServerVersion)).
		GET("site-settings", UpButterReq(api.GetSiteSettings)).
		POST("save-site-settings", UpButterReq(api.SaveSiteSettings)).
		GET("site-chrome", UpButterReq(api.GetSiteChrome)).
		POST("save-site-chrome", UpButterReq(api.SaveSiteChrome)).
		GET("site-theme", UpButterReq(api.GetSiteTheme)).
		POST("save-site-theme", UpButterReq(api.SaveSiteTheme)).
		POST("publish-site-theme", UpButterReq(api.PublishSiteTheme)).
		GET("mail-settings", UpButterReq(api.GetMailSettings)).
		POST("save-mail-settings", UpButterReq(api.SaveMailSettings)).
		POST("test-mail-connection", UpButterReq(api.TestMailConnection)).
		GET("security-settings", UpButterReq(api.GetSecuritySettings)).
		POST("save-security-settings", UpButterReq(api.SaveSecuritySettings)).
		GET("posting-settings", UpButterReq(api.GetPostingSettings)).
		POST("save-posting-settings", UpButterReq(api.SavePostingSettings)).
		GET("rate-limit-settings", UpButterReq(api.GetRateLimitSettings)).
		POST("save-rate-limit-settings", UpButterReq(api.SaveRateLimitSettings)).
		GET("storage-settings", UpButterReq(api.GetStorageSettings)).
		POST("save-storage-settings", UpButterReq(api.SaveStorageSettings)).
		POST("test-storage-connection", UpButterReq(api.TestStorageConnection)).
		POST("storage-migrate-task", UpButterReq(api.CreateStorageMigrateTask)).
		GET("storage-migrate-tasks", UpButterReq(api.GetStorageMigrateTasks)).
		GET("http-notify-settings", UpButterReq(api.GetHttpNotifySettings)).
		POST("save-http-notify-settings", UpButterReq(api.SaveHttpNotifySettings)).
		GET("badges", UpButterReq(api.BadgeList)).
		POST("badge-save", UpButterReq(api.SaveBadge)).
		POST("badge-delete", UpButterReq(api.DeleteBadge)).
		GET("terms-of-service", UpButterReq(api.GetTermsOfService)).
		POST("save-terms-of-service", UpButterReq(api.SaveTermsOfService)).
		POST("file-resources", UpButterReq(api.FileResourcePage)).
		POST("img-upload", api.SaveAdminImgByGinContext).
		POST("data/export", UpButterReq(api.CreateExportTask)).
		GET("data/export/tasks", UpButterReq(api.ListExportTasks)).
		GET("data/export/download/:taskId", api.DownloadExportTask).
		POST("data/import", api.ImportData).
		POST("review-queue", UpButterReq(api.ReviewQueue)).
		POST("review-action", UpButterReq(api.ReviewAction))

}

func fileServer(ginApp *gin.Engine) {
	r := ginApp.Group("file")
	r.POST("/img-upload", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.SaveImgByGinContext)
	r.GET("/img/*filename", api.GetFileByFileName)
}
