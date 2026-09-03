package middleware

import (
	"net/http"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/gin-gonic/gin"
)

// Header keys set by the security middleware.
const (
	headerXContentTypeOptions   = "X-Content-Type-Options"
	headerXFrameOptions         = "X-Frame-Options"
	headerReferrerPolicy        = "Referrer-Policy"
	headerPermissionsPolicy     = "Permissions-Policy"
	headerContentSecurityPolicy = "Content-Security-Policy"
)

// SecurityHeaders 为全站响应统一附加安全响应头，并按「是否为 HTML 页面路由」决定是否附加页面级 CSP。
//
// 分层策略：
//   - 所有响应（HTML 页面 / API JSON / 图片与对象存储下载 / 静态资源 / 错误页共用本中间件）：
//     X-Content-Type-Options: nosniff、X-Frame-Options: DENY（旧浏览器点击劫持兜底）、
//     Referrer-Policy: strict-origin-when-cross-origin、Permissions-Policy: camera/microphone/geolocation 全禁。
//     X-Frame-Options 对非文档响应（JSON/图片）无副作用；nosniff 是下载型响应的 MIME 防混淆基线。
//   - HTML 页面路由（forum viewRoute 等引擎级 GET，含 404/500 页面渲染）额外附加 Content-Security-Policy，
//     frame-ancestors 'none' 与 X-Frame-Options: DENY 组成点击劫持双保险（全仓前端无任何 <iframe> 用法）。
//   - /api /file /mcp /assets /static 等非页面表面不加 CSP：浏览器只对文档响应执行 CSP，
//     附加到 JSON/图片上无意义；OIDC 库自渲染的 authorize 页面位于 /api/oauth/* 之下，
//     不受本策略约束（其交互不依赖第三方源）。
//
// 注册顺序约束（见 bridge.go）：本中间件注册在 SiteMaintenance 之后。维护模式/启动门响应
// 自带内联样式与脚本（如维护页时钟、自动刷新），若先于它们注册，页面级 CSP（script-src 'self'）
// 会禁掉这些脚本；因此这两个临时静态页响应不附加安全头——它们无会话、无交互状态，
// 点击劫持面可忽略。一旦服务就绪（维护关闭、启动门放行），所有后续请求都会经过本中间件。
func SecurityHeaders(c *gin.Context) {
	headers := c.Writer.Header()
	headers.Set(headerXContentTypeOptions, "nosniff")
	headers.Set(headerXFrameOptions, "DENY")
	headers.Set(headerReferrerPolicy, "strict-origin-when-cross-origin")
	headers.Set(headerPermissionsPolicy, "camera=(), microphone=(), geolocation=()")
	if isHTMLPageRoute(c) {
		// 页面控制器渲染模板前不写 Content-Type，响应提交后再补头对已发送响应无效，
		// 因此必须在 c.Next() 之前按路由形态决策（与响应状态无关，404/500 页面同样覆盖）。
		headers.Set(headerContentSecurityPolicy, buildPageCSP(setting.IsProduction()))
	}
	c.Next()
}

// isHTMLPageRoute 按请求路由形态判断该响应是否承载浏览器 HTML 文档。
// 排除非页面表面（API/OIDC/文件/静态资源/健康检查）；其余引擎级 GET 属于 forum viewRoute
// 页面、/activate 或 robots/sitemap/rss/llms 等根级文本导出——对非文档响应浏览器忽略 CSP，
// frame-ancestors 仍然生效。新增页面路由无需改这里；新增非页面根级路由时请在此显式排除。
func isHTMLPageRoute(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}
	path := c.Request.URL.Path
	switch {
	case path == "/health", path == "/reload", path == "/mcp":
		return false
	case strings.HasPrefix(path, "/api/"), strings.HasPrefix(path, "/file/"),
		strings.HasPrefix(path, "/assets/"), strings.HasPrefix(path, "/static/"):
		return false
	}
	return true
}

// buildPageCSP 生成页面级 Content-Security-Policy。
//
// 收紧（XSS 主防线）：script-src 'self'（构建产物无内联执行脚本、无 eval，模板中的
// application/json 数据块不参与脚本执行）、object-src 'none'、frame-src 'none'、
// frame-ancestors 'none'、base-uri 'self'、form-action 'self'。刻意不使用 report-only：
// 头已按现有模板/前端实际源形态审计过，report-only 不作为交付形态。
//
// 已审计的放宽项（均有实际依据，不涉及 script-src）：
//   - style-src 'unsafe-inline'：Vue 动态 :style 绑定、GoHTML 内联 <style>（noscript 爬虫样式、
//     activate 页）都需要内联样式；'unsafe-inline' 限 style-src，不影响脚本防线。
//   - img-src data: blob:：头像/封面裁剪预览（URL.createObjectURL）与课程分享图（data:image/svg）。
//   - style-src/font-src 放行 fonts.googleapis.cn / fonts.gstatic.cn：站点头部 Noto Serif SC
//     走该国内镜像（site/main.ts installNotoSerifSc）。
//   - connect-src http: https:：对象存储浏览器直传指向可配置的预签名端点（S3/OSS/COS/MinIO，
//     管理端另请求 api.github.com），端点主机随部署配置变化，只能按 scheme 放行；
//     https 站点下 http 目标本就会被浏览器混合内容策略拦截，放行 http: 仅保住 http 内网部署。
//   - 非生产（本地 dev）额外放行 ws: wss:：Vite HMR WebSocket（经同源 /assets 代理后仍按 ws: 连接）。
//
// 已知保留风险（有意不为此放宽）：
//   - Vditor 编辑器按官方 CDN 路径维护的 highlight 样式 <link> 属预期的 404 请求，样式由本地
//     主题 link 提供，被 style-src 拦截无功能影响；mathjax/echarts CDN 脚本同样被 script-src 'self'
//     拦截（本地已用 katex/echarts 同源资源替代，见 VditorOfficial.vue 预置说明）。
func buildPageCSP(production bool) string {
	connectSrc := "'self' http: https:"
	if !production {
		connectSrc += " ws: wss:"
	}
	return "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.cn; " +
		"font-src 'self' https://fonts.gstatic.cn; " +
		"img-src 'self' http: https: data: blob:; " +
		"connect-src " + connectSrc + "; " +
		"media-src 'self'; " +
		"frame-src 'none'; " +
		"object-src 'none'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"
}
