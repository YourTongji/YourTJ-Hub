package oidcservice

import "net/http"

// withDefaultContentType 复刻 net/http 的默认内容嗅探：handler 未声明
// Content-Type 且写出 body 时，按首块字节的嗅探结果补写 Content-Type。
//
// 背景：authorize 的 form_post HTML 由 zitadel/oidc 直接执行模板写出，只设
// Cache-Control 不写 Content-Type（库 auth_request.go 的 AuthResponseFormPost）。
// 目前 /api/oauth 不在 gzip 路由组，缺失的 Content-Type 由 net/http 嗅探兜底；
// 一旦该表面将来挂入压缩路由组，压缩 writer 会在首字节提交响应头，嗅探不再
// 发生，叠加 X-Content-Type-Options: nosniff 会把页面按 text/plain 展示源码
// （view 路由已在 #441 修复同类缺陷，此处为对称防护）。本包装使行为与
// 无压缩时的 net/http 完全一致，压缩与否不再影响该表面。
func withDefaultContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Content-Type") == "" {
			w = &defaultContentTypeWriter{ResponseWriter: w}
		}
		next.ServeHTTP(w, r)
	})
}

// defaultContentTypeWriter 仅在首个 body Write 时兜底补头；302 等无 body
// 响应不经过 Write，不会多出 Content-Type。已显式声明 Content-Type 的
// 响应（如 discovery 的 application/json）不做任何改写。
type defaultContentTypeWriter struct {
	http.ResponseWriter
	sniffed bool
}

func (w *defaultContentTypeWriter) Write(p []byte) (int, error) {
	if !w.sniffed {
		w.sniffed = true
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(p))
		}
	}
	return w.ResponseWriter.Write(p)
}

// Flush 透传，避免包装丢失 http.Flusher 能力导致下游类型断言失败。
// 首字节补头发生在 Write 中，Flush 语义不受影响。
func (w *defaultContentTypeWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
