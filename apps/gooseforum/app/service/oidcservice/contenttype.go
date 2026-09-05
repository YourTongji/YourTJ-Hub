package oidcservice

import "net/http"

const authorizeHTMLContentType = "text/html; charset=utf-8"

// withDefaultHTMLContentType 在请求进入下游前为 authorize 端点预置 HTML
// Content-Type，下游显式声明时以其为准（Header.Set 覆盖同 key）。
//
// 为什么在入口预置而不是写出阶段补嗅探：authorize 的 form_post HTML 由
// zitadel/oidc 直接写出（AuthResponseFormPost 先 WriteHeader(200) 再写模板，
// 且此前已设置 Cache-Control）；真实 net/http 在 handler 触碰过 Header 后
// 调用 WriteHeader 的瞬间即克隆提交响应头，写出阶段的补写对已提交的头不再
// 生效（httptest.ResponseRecorder 共享 header map 会掩盖该时序，见 #441
// review）。入口预置使该表面与是否挂入 gzip 路由组解耦——view 路由已在
// #441 修复同类缺陷，此为对称防护。
//
// 未登录 authorize 的 302 重定向分支也会携带该头：与 net/http http.Redirect
// 对 GET 重定向显式写 text/html 的标准行为一致，客户端不渲染无 body 响应，
// 无实际影响。
func withDefaultHTMLContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", authorizeHTMLContentType)
		}
		next.ServeHTTP(w, r)
	})
}
