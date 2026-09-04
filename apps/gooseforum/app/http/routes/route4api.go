package routes

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/gin-contrib/gzip"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oidcservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/resource"
	"github.com/gin-gonic/gin"
)

func gzipEnabled() bool {
	return preferences.GetBool("server.gzip", true)
}

func assertRouter(ginApp *gin.Engine) {
	assetsFs, _ := resource.GetAssetsFS()
	staticFS, _ := resource.GetStaticFS()
	staticRoute := ginApp.Group("/")
	if gzipEnabled() {
		staticRoute.Use(gzip.Gzip(gzip.DefaultCompression))
		slog.Info("static assets gzip enabled")
	} else {
		slog.Info("static assets gzip disabled")
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
	viewRouteApp.GET("/topics/:id", forum.TopicDetail)
	viewRouteApp.GET("/topics/:id/:postNo", forum.TopicDetail)
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
	viewRouteApp.GET("/wiki", forum.WikiHome)
	viewRouteApp.GET("/wiki/*path", forum.WikiDetail)
	viewRouteApp.GET("/courses", middleware.RateLimit(middleware.RateLimitCourseCatalog), forum.CourseCatalog)
	viewRouteApp.GET("/courses/:courseId", middleware.RateLimit(middleware.RateLimitCourseCatalog), forum.CourseDetail)
	viewRouteApp.GET("/moderation/course-reviews", middleware.CheckLogin, forum.CourseReviewModeration)
	viewRouteApp.GET("/moderation/courses", middleware.CheckLogin, forum.CourseManagement)
	viewRouteApp.GET("/schedule", middleware.RateLimit(middleware.RateLimitCourseCatalog), forum.Schedule)
	viewRouteApp.GET("/admin", middleware.CheckLogin, middleware.CheckAnyPermissionOrNotFound, forum.Manage)
	viewRouteApp.GET("/admin/*path", middleware.CheckLogin, middleware.CheckAnyPermissionOrNotFound, forum.Manage)
	viewRouteApp.GET("/login", forum.Login)
	viewRouteApp.GET("/reset-password", forum.ResetPassword)
	viewRouteApp.GET("/terms", forum.Terms)
	viewRouteApp.GET("/privacy", forum.Privacy)

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
	baseApi.POST("logout", middleware.CSRFProtection, api.Logout)

	baseApi.GET("get-captcha", UpQueryReq(api.GetCaptcha))
	baseApi.GET("user-card", UpQueryReq(api.GetUserCard))
	baseApi.POST("forgot-password", middleware.RateLimit(middleware.RateLimitForgotPassword), UpButterReq(api.ForgotPassword))
	baseApi.POST("reset-password", middleware.RateLimit(middleware.RateLimitResetPassword), UpButterReq(api.ResetPassword))
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

	// CSRF 防护（issue #406）：挂在认证之后的写组中间件，仅校验「Cookie 可认证 +
	// 状态变更方法」请求（Bearer 客户端豁免），详见 middleware/csrfProtection.go
	// 与 docs/product/identity-and-access.md「认证与 CSRF 边界」。不挂全局，避免与
	// #407 全局安全头中间件冲突；GET/HEAD/OPTIONS 与匿名请求原样放行。
	loginApi := ginApp.Group("api").Use(middleware.JWTAuthCheck, middleware.CSRFProtection)
	loginApi.POST("set-user-info", middleware.CheckWritableAccount, UpButterReq(api.EditUserInfo))
	loginApi.POST("set-user-profile-cover", middleware.CheckWritableAccount, UpButterReq(api.EditUserProfileCover))
	loginApi.POST("set-user-email", middleware.CheckWritableAccountAllowPendingActivation, middleware.RateLimit(middleware.RateLimitEmailChange), UpButterReq(api.EditUserEmail))
	loginApi.POST("resend-activation-email", middleware.CheckWritableAccountAllowPendingActivation, UpButterReq(api.ResendActivationEmail))
	loginApi.POST("set-user-name", middleware.CheckWritableAccount, UpButterReq(api.EditUsername))
	loginApi.POST("set-preset-avatar", middleware.CheckWritableAccount, UpButterReq(api.SetPresetAvatar))
	loginApi.POST("wear-badge", middleware.CheckWritableAccount, UpButterReq(api.WearBadge))
	loginApi.POST("upload-avatar", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.UploadAvatar)
	loginApi.POST("change-password", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPasswordChange), UpButterReq(api.ChangePassword))
	loginApi.POST("auth/:provider/unbind", middleware.CheckWritableAccount, UpButterReq(api.UnbindOAuth))
	loginApi.GET("oauth/bindings", UpButterReq(api.GetOAuthBindings))
	loginApi.GET("user/sessions", UpButterReq(api.ListSessions))
	loginApi.POST("user/sessions/revoke", UpButterReq(api.RevokeSession))
	loginApi.POST("user/sessions/revoke-all", UpButterReq(api.RevokeAllSessions))
	// TOTP 写操作校验账户密码或 6 位验证码（setup/enable/disable），挂 RateLimit 防止暴力破解；
	// status 只读 enabled 标志、不验证任何凭据，无需限流。
	loginApi.POST("user/totp/setup", middleware.RateLimit(middleware.RateLimitTotpSetup), UpButterReq(api.TotpSetup))
	loginApi.POST("user/totp/enable", middleware.RateLimit(middleware.RateLimitTotpEnable), UpButterReq(api.TotpEnable))
	loginApi.POST("user/totp/disable", middleware.RateLimit(middleware.RateLimitTotpDisable), UpButterReq(api.TotpDisable))
	loginApi.GET("user/totp/status", UpButterReq(api.TotpStatus))

	// PK 排课器 13 端点（Issue #187）：公开只读，统一 {code,msg,data} 信封，
	// 与现有 /api/forum 的 {result,code,messageCode,params} 信封并列。
	// 限流复用课程目录配额（只读目录类访问）。
	pkApi := baseApi.Group("pk")
	pkApi.GET("calendars", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkNoReq(pkcontroller.ListCalendars))
	pkApi.GET("campuses", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkNoReq(pkcontroller.ListCampuses))
	pkApi.GET("faculties", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkNoReq(pkcontroller.ListFaculties))
	pkApi.POST("grades", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.Grades))
	pkApi.POST("majors", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.Majors))
	pkApi.POST("courses-by-major", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.CoursesByMajor))
	pkApi.POST("optional-types", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.OptionalTypes))
	pkApi.POST("courses-by-nature", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.CoursesByNature))
	pkApi.POST("course-details", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.CourseDetails))
	pkApi.POST("course-search", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.CourseSearch))
	pkApi.POST("courses-by-time", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.CoursesByTime))
	pkApi.GET("latest-update", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkNoReq(pkcontroller.LatestUpdate))
	pkApi.POST("course-info-sync", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkJsonReq(pkcontroller.CourseInfoSync))
	pkApi.GET("course-review-brief", middleware.RateLimit(middleware.RateLimitCourseCatalog), pkQueryReq(pkcontroller.CourseReviewBrief))

	forumApi := baseApi.Group("forum")
	forumApi.GET("get-site-statistics", ginUpNP(api.GetSiteStatistics))
	forumApi.GET("search", middleware.JWTAuth, UpQueryReq(forum.SearchJSON))
	forumApi.GET("courses", middleware.RateLimit(middleware.RateLimitCourseCatalog), UpQueryReq(forum.CourseListJSON))
	forumApi.GET("courses/:courseId", middleware.RateLimit(middleware.RateLimitCourseCatalog), UpUriQueryReq(forum.CourseDetailJSON))
	// 课程评价列表：公开可读，可选 JWT 仅用于 viewer 状态，不要求登录。
	forumApi.GET("courses/:courseId/reviews", middleware.RateLimit(middleware.RateLimitCourseCatalog), middleware.JWTAuth, UpUriQueryReq(forum.ListCourseReviews))
	// 相关课程：同教师其他课 + 同课程其他教师（公开只读，与课程目录共用限流配额）。
	forumApi.GET("courses/:courseId/related", middleware.RateLimit(middleware.RateLimitCourseCatalog), UpUriQueryReq(forum.CourseRelatedJSON))
	// wiki 分站：公开读。
	// wiki 分站：公开读（GitHub SSOT：内容由仓库同步，无站内写）。
	wikiApi := baseApi.Group("wiki")
	wikiApi.GET("tree", UpButterReq(api.WikiTree))
	wikiApi.GET("namespaces", UpButterReq(api.WikiNamespaces))
	wikiApi.GET("home", UpButterReq(api.WikiHome))
	// wiki 站内局内搜索（前端搜索面板；段落级 Meilisearch 索引，公开只读）。
	wikiApi.GET("search", UpQueryReq(forum.WikiSearchJSON))
	// wiki GitHub webhook：PR merge 后即时同步（独立验签，无 JWT）。
	// 公开端点加限流（review MEDIUM）：防未认证调用方以超大 body 刷 HMAC
	// 计算（CPU DoS）与重放触发全量同步。
	wikiApi.POST("webhook", middleware.RateLimit(middleware.RateLimitWikiWebhook), api.WikiWebhook)
	// 课程 AI 总结（B7, issue #181）：公开只读；可选 JWT 先于 RateLimit 解析
	// 用户身份（course.summary 的 limitPerUser / skipAdmin 依赖 userId），
	// 未登录调用者仍可读（JWTAuth 可选）。
	// check 预检（?check=true）走独立 course.summary.check 配额，不消耗生成配额
	// （review P2：浏览 N 门课程页的挂载预检不得耗尽 per-User 生成配额）。
	forumApi.GET("courses/:courseId/summary", middleware.JWTAuth, middleware.RateLimitCourseSummaryAware(), UpUriQueryReq(forum.GetCourseSummary))
	forumApi.GET("posts/window", middleware.JWTAuth, middleware.NoUpdateUserActivity, UpQueryReq(forum.PostWindow))
	// 帖子版本历史：公开只读（话题可见即可读），可选 JWT 仅用于 viewer 状态，
	// 不要求登录；待审版本正文在控制器内对非版主屏蔽。
	forumApi.GET("posts/revisions", middleware.JWTAuth, middleware.NoUpdateUserActivity, UpQueryReq(forum.PostRevisions))

	forumLoginApi := forumApi.Use(middleware.JWTAuthCheck, middleware.CSRFProtection)
	forumLoginApi.GET("unread-status", middleware.NoUpdateUserActivity, UpButterReq(api.GetUnreadStatus))
	forumLoginApi.GET("notifications", middleware.NoUpdateUserActivity, UpQueryReq(api.NotificationList))
	forumLoginApi.POST("notification/mark-read", middleware.CheckWritableAccount, UpButterReq(api.MarkAsRead))
	forumLoginApi.POST("notification/mark-all-read", middleware.CheckWritableAccount, UpButterReq(api.MarkAllAsRead))
	forumLoginApi.POST("topics/write", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitTopicWrite), UpButterReq(api.WriteTopic))
	forumLoginApi.POST("topics/status", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitTopicStatus), UpButterReq(api.UpdateTopicStatus))
	forumLoginApi.POST("topics/delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.DeleteTopicByUser))
	forumLoginApi.GET("user/deleted-content", middleware.NoUpdateUserActivity, UpQueryReq(api.DeletedContentList))
	forumLoginApi.GET("user/my-content", middleware.NoUpdateUserActivity, UpQueryReq(api.MyContentList))
	forumLoginApi.POST("user/content-batch-delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BatchDeleteContent))
	forumLoginApi.POST("user/content-restore", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.RestoreContent))
	forumLoginApi.POST("user/content-purge", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.PurgeContent))
	forumLoginApi.POST("user/content-privacy-erase", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.PrivacyErase))
	forumLoginApi.POST("user/content-event", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.ReportContentEvent))
	forumLoginApi.POST("user/account-close", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AccountClose))
	forumLoginApi.POST("posts/create", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostCreate), UpButterReq(api.CreatePost))
	forumLoginApi.POST("posts/update", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostUpdate), UpButterReq(api.UpdatePost))
	forumLoginApi.POST("posts/delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostDelete), UpButterReq(api.DeletePost))
	forumLoginApi.POST("posts/like", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.LikePost))
	forumLoginApi.POST("posts/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BookmarkPost))
	forumLoginApi.POST("topics/like", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.LikeTopic))
	forumLoginApi.POST("topics/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BookmarkTopic))
	forumLoginApi.POST("topics/watch", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.WatchTopic))
	forumLoginApi.POST("follow-user", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.FollowUser))
	forumLoginApi.POST("report", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(forum.CreateReport))
	// 课评写接口：登录 + 可写账号 + 独立限流。
	forumLoginApi.POST("course-reviews", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewWrite), UpJsonReq(forum.CreateCourseReview))
	forumLoginApi.PATCH("course-reviews/:reviewId", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewWrite), UpUriJsonReq(forum.UpdateCourseReview))
	forumLoginApi.DELETE("course-reviews/:reviewId", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewWrite), UpUriReq(forum.DeleteCourseReview))
	forumLoginApi.PUT("course-reviews/:reviewId/helpful", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewHelpful), UpUriReq(forum.MarkReviewHelpful))
	forumLoginApi.DELETE("course-reviews/:reviewId/helpful", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewHelpful), UpUriReq(forum.UnmarkReviewHelpful))
	forumLoginApi.PUT("course-reviews/:reviewId/dislike", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewDislike), UpUriReq(forum.MarkReviewDislike))
	forumLoginApi.DELETE("course-reviews/:reviewId/dislike", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewDislike), UpUriReq(forum.UnmarkReviewDislike))
	forumLoginApi.POST("course-reviews/:reviewId/reports", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewReport), UpUriJsonReq(forum.ReportCourseReview))
	// 课程收藏：登录 + 可写账号 + 独立限流（对齐 topics/bookmark 的 action 1/2 幂等）。
	forumLoginApi.POST("courses/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitCourseBookmark), UpButterReq(forum.BookmarkCourse))
	// 课评审核：独立 CourseManager 权限；身份揭示仅 Admin（控制器内二次校验）。
	// 审核操作挂 course.review.moderate 限流（60s per-IP 60 / per-User 30，issue #176 B4）。
	// 权限校验前置到 RateLimit 之前：未授权请求直接 403，不消耗共享 per-IP 配额
	// （否则任意登录用户可耗尽同 IP 审核员的限流池，DoS 审核功能，security review F1）。
	forumLoginApi.POST("moderation/course-review-status", middleware.CheckWritableAccount, middleware.CheckPermission(permission.CourseManager), middleware.RateLimit(middleware.RateLimitReviewModerate), UpButterReq(forum.ModerationCourseReviewStatus))
	forumLoginApi.POST("moderation/course-review-reports", middleware.NoUpdateUserActivity, middleware.CheckPermission(permission.CourseManager), middleware.RateLimit(middleware.RateLimitReviewModerate), UpButterReq(forum.ModerationCourseReviewReportList))
	forumLoginApi.POST("moderation/course-review-reveal", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewReveal), UpButterReq(forum.ModerationCourseReviewReveal))
	// 课评管理：课程/评价 CRUD + 统计重建（CourseManager 权限，控制器内校验）。
	forumLoginApi.POST("moderation/course-list", middleware.NoUpdateUserActivity, UpButterReq(forum.AdminCourseList))
	forumLoginApi.POST("moderation/course-create", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseCreate))
	forumLoginApi.POST("moderation/course-update", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseUpdate))
	forumLoginApi.POST("moderation/course-delete", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseDelete))
	forumLoginApi.POST("moderation/course-review-list", middleware.NoUpdateUserActivity, UpButterReq(forum.AdminReviewList))
	forumLoginApi.POST("moderation/course-review-edit", middleware.CheckWritableAccount, UpButterReq(forum.AdminReviewUpdate))
	forumLoginApi.POST("moderation/course-review-delete", middleware.CheckWritableAccount, UpButterReq(forum.AdminReviewDelete))
	forumLoginApi.POST("moderation/course-stats-rebuild", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseStatsRebuild))
	forumLoginApi.POST("moderation/course-relation-list", middleware.NoUpdateUserActivity, UpButterReq(forum.AdminCourseRelationList))
	forumLoginApi.POST("moderation/course-relation-approve", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseRelationApprove))
	forumLoginApi.POST("moderation/course-relation-ignore", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseRelationIgnore))
	forumLoginApi.POST("moderation/course-relation-create", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseRelationCreate))
	forumLoginApi.POST("moderation/course-relation-reset", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseRelationReset))
	forumLoginApi.POST("moderation/course-merge", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseMerge))
	forumLoginApi.POST("moderation/course-merge-undo", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseMergeUndo))
	forumLoginApi.POST("moderation/topic-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationTopicStatus))
	forumLoginApi.POST("moderation/post-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationPostStatus))
	forumLoginApi.POST("moderation/reports", middleware.NoUpdateUserActivity, UpButterReq(forum.ModerationReportList))
	forumLoginApi.POST("moderation/report-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationReportStatus))
	forumLoginApi.POST("moderation/logs", middleware.NoUpdateUserActivity, UpButterReq(forum.ModerationLogList))
	forumLoginApi.POST("moderation/view-deleted-content", middleware.CheckWritableAccount, UpButterReq(forum.ViewDeletedContent))

	chatApi := forumApi.Group("chat", middleware.JWTAuthCheck, middleware.CSRFProtection)

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

	adminApi := baseApi.Group("admin", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.CSRFProtection)

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
		POST("topics/restore", UpButterReq(api.RestoreTopic)).
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
		POST("save-announcement", UpButterReq(api.SaveAnnouncement)).
		GET("wiki/tree", UpButterReq(api.WikiAdminTree)).
		GET("wiki/sync/status", UpButterReq(api.WikiSyncStatus)).
		POST("wiki/sync", UpButterReq(api.WikiSyncRun)).
		GET("wiki/sync/runs", UpButterReq(api.WikiSyncRuns)).
		GET("wiki/sync/webhook-secret", UpButterReq(api.GetWikiWebhookSecret)).
		POST("wiki/sync/webhook-secret", UpJsonReq(api.SaveWikiWebhookSecret)).
		GET("wiki/sync/cdn", UpButterReq(api.GetWikiAssetCDN)).
		POST("wiki/sync/cdn", UpJsonReq(api.SaveWikiAssetCDN))

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
		GET("mcp-settings", UpButterReq(api.GetMCPSettings)).
		POST("save-mcp-settings", UpButterReq(api.SaveMCPSettings)).
		GET("schedule-settings", UpButterReq(api.GetScheduleSettings)).
		POST("save-schedule-settings", UpButterReq(api.SaveScheduleSettings)).
		GET("onesystem-settings", UpButterReq(api.GetOnesystemSettings)).
		POST("save-onesystem-settings", UpButterReq(api.SaveOnesystemSettings)).
		// 排课数据同步（issue #248 自愈入口）：触发同步 + 查询各学期状态。
		POST("pk/sync-calendar", UpButterReq(api.SyncPkCalendar)).
		GET("pk/sync-status", UpButterReq(api.PkSyncStatus)).
		GET("ai-summary-settings", UpButterReq(api.GetAiSummarySettings)).
		POST("save-ai-summary-settings", UpButterReq(api.SaveAiSummarySettings)).
		POST("ai-summary-models", UpButterReq(api.ListAiSummaryModels)).
		POST("badge-save", UpButterReq(api.SaveBadge)).
		POST("badge-delete", UpButterReq(api.DeleteBadge)).
		GET("terms-of-service", UpButterReq(api.GetTermsOfService)).
		POST("save-terms-of-service", UpButterReq(api.SaveTermsOfService)).
		GET("privacy-policy", UpButterReq(api.GetPrivacyPolicy)).
		POST("save-privacy-policy", UpButterReq(api.SavePrivacyPolicy)).
		POST("file-resources", UpButterReq(api.FileResourcePage)).
		POST("img-upload", api.SaveAdminImgByGinContext).
		POST("data/export", UpButterReq(api.CreateExportTask)).
		GET("data/export/tasks", UpButterReq(api.ListExportTasks)).
		GET("data/export/download/:taskId", api.DownloadExportTask).
		POST("data/import", api.ImportData).
		GET("data/import/tasks", UpButterReq(api.ListImportTasks)).
		POST("data/import/tasks/:taskId/replay", UpUriReq(api.ReplayImportTask)).
		POST("review-queue", UpButterReq(api.ReviewQueue)).
		POST("review-action", UpButterReq(api.ReviewAction))

}

func fileServer(ginApp *gin.Engine) {
	r := ginApp.Group("file", middleware.CSRFProtection)
	r.POST("/img-upload", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.SaveImgByGinContext)
	r.POST("/img-upload/init", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.InitDirectImageUpload)
	r.POST("/img-upload/complete", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.CompleteDirectImageUpload)
	r.POST("/img-upload/abort", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitUpload), api.AbortDirectImageUpload)
	r.GET("/img/*filename", api.GetFileByFileName)
}
