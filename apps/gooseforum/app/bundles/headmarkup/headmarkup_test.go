package headmarkup

import (
	"strings"
	"testing"
)

// issueExternalLinks 是 issue #1244 中站点设置里实际生效的那段整段 <head> payload。
const issueExternalLinks = "<head>\n" +
	"    <meta charset=\"UTF-8\">\n" +
	"    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n" +
	"    <title>YourTJ社区</title>\n" +
	"    <meta name=\"description\" content=\"同济大学学生社区\">\n" +
	"    <meta name=\"keywords\" content=\"forum,discussion,community,YourTJ,同济大学,YourTJ社区\">\n" +
	"    <meta name=\"robots\" content=\"index, follow\">\n" +
	"    <meta name=\"author\" content=\"YourTJ Community\">\n" +
	"</head>\n"

func TestSanitize(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "issue payload keeps only the custom metas",
			raw:  issueExternalLinks,
			want: "<meta name=\"keywords\" content=\"forum,discussion,community,YourTJ,同济大学,YourTJ社区\"/>\n" +
				"<meta name=\"author\" content=\"YourTJ Community\"/>",
		},
		{
			name: "nested document drops structure and shell-owned tags",
			raw: "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">" +
				"<meta name=\"viewport\" content=\"width=device-width\">" +
				"<meta name=\"description\" content=\"dup\">" +
				"<meta name=\"robots\" content=\"index, follow\">" +
				"<title>dup</title>" +
				"<meta name=\"keywords\" content=\"k\">" +
				"<meta name=\"author\" content=\"a\">" +
				"</head><body><script>alert(1)</script>body text</body></html>",
			want: "<meta name=\"keywords\" content=\"k\"/>\n<meta name=\"author\" content=\"a\"/>",
		},
		{
			name: "plain head fragment keeps shell-owned tags out",
			raw:  "<meta name=\"viewport\" content=\"width=device-width\">\n<meta name=\"theme-color\" content=\"#123456\">",
			want: "",
		},
		{
			name: "uppercase tags and attribute names",
			raw: "<HEAD><META CHARSET=\"utf-8\"><META NAME=\"ViewPort\" CONTENT=\"width=device-width\">" +
				"<TITLE>dup</TITLE><META NAME=\"Keywords\" CONTENT=\"K\"></HEAD>",
			// 标签与属性名由解析器小写化，属性值原样保留。
			want: "<meta name=\"Keywords\" content=\"K\"/>",
		},
		{
			// 外壳自己的 theme-color 与 canonical 同样属于每页文档元数据。
			name: "shell-owned theme-color and canonical are stripped",
			raw: `<meta name="theme-color" content="#123456">` + "\n" +
				`<link rel="canonical" href="https://example.test/">`,
			want: "",
		},
		{
			// og:/twitter: 前缀的 meta 由页面按自己的 SEO 取值输出，属性名大小写
			// 与取值两侧空白都不影响判定。
			name: "og and twitter metas are stripped from name or property",
			raw: `<meta property="OG:title" content="dup">` + "\n" +
				`<meta name=" Twitter:card " content="summary">` + "\n" +
				`<link rel=" Canonical " href="https://example.test/">`,
			want: "",
		},
		{
			// article:* 由话题页的 og 元数据输出；name 与 property 两种写法都剥离。
			name: "article metas are stripped from name or property",
			raw: `<meta property="article:published_time" content="2024-01-01T00:00:00Z">` + "\n" +
				`<meta name="Article:Tag" content="go">`,
			want: "",
		},
		{
			// rel 是 token 集合：含外壳 link 的 token 即整条剥离（canonical
			// alternate、shortcut icon），纯 alternate 仍保留。strings.Fields 已处理
			// 多余空白。
			name: "rel tokens strip shell-owned relations and keep alternate",
			raw: `<link rel="canonical alternate" href="https://example.test/">` + "\n" +
				`<link rel="alternate" href="https://example.test/feed.atom">` + "\n" +
				`<link rel="shortcut icon" href="/custom.ico">`,
			want: `<link rel="alternate" href="https://example.test/feed.atom"/>`,
		},
		{
			// 其余自定义 link 与 meta 是该字段的正当内容，必须保留。
			name: "custom alternate link and keywords meta survive",
			raw: `<link rel="alternate" href="https://example.test/feed.atom">` + "\n" +
				`<meta name="keywords" content="k">`,
			want: `<link rel="alternate" href="https://example.test/feed.atom"/>` + "\n" +
				`<meta name="keywords" content="k"/>`,
		},
		{
			// 实际存在的失败路径：head 内合法元素嵌套超过 512 层时 html.Parse 内部
			// panic 并返回错误，Sanitize fail-closed 返回空串。嵌套 <template> 让该
			// 断言区分失败分支与「内容本就在 body 被丢弃」：浅层嵌套会返回标记。
			name: "over-deeply nested head input fails closed",
			raw:  strings.Repeat("<template>", 600) + `<meta name="keywords" content="k">` + strings.Repeat("</template>", 600),
			want: "",
		},
		{
			// 同一结构与上一个用例的对照组：浅层嵌套会返回标记，说明分界在解析失败
			// 而不是该结构被剥离。
			name: "shallow nested head elements are kept",
			raw:  strings.Repeat("<template>", 3) + `<meta name="keywords" content="k">` + strings.Repeat("</template>", 3),
			want: `<template><template><template><meta name="keywords" content="k"/></template></template></template>`,
		},
		{
			// 无值布尔属性经 x/net/html 序列化为 attr=""（HTML 中语义相同）。
			name: "script link style noscript and meta survive",
			raw: "<script defer src=\"/custom.js\"></script>\n" +
				"<link rel=\"preconnect\" href=\"https://cdn.example.com\">\n" +
				"<style>body{color:red}</style>\n" +
				"<noscript><img src=\"/x.png\"></noscript>\n" +
				"<meta name=\"keywords\" content=\"k\">",
			want: "<script defer=\"\" src=\"/custom.js\"></script>\n" +
				"<link rel=\"preconnect\" href=\"https://cdn.example.com\"/>\n" +
				"<style>body{color:red}</style>\n" +
				"<noscript><img src=\"/x.png\"></noscript>\n" +
				"<meta name=\"keywords\" content=\"k\"/>",
		},
		{
			name: "attribute values are not escaped twice",
			raw:  `<meta name="custom" content="a &amp; b &lt;c&gt;">`,
			want: `<meta name="custom" content="a &amp; b &lt;c&gt;"/>`,
		},
		{
			// http-equiv=content-type 与外壳的 <meta charset> 是同一类重复声明；
			// 其它 http-equiv 策略（CSP 等）照旧保留。
			name: "content-type http-equiv is shell-owned others pass through",
			raw: `<meta http-equiv="Content-Type" content="text/html; charset=utf-8">` + "\n" +
				`<meta http-equiv="content-security-policy" content="default-src 'self'">`,
			// 属性值经 html.Render 转义：单引号写成 &#39;（浏览器解码后等价）。
			want: `<meta http-equiv="content-security-policy" content="default-src &#39;self&#39;"/>`,
		},
		{
			// x/net/html 把 <body> 内的 head 风格元素也插在 body 里（不回插 head），
			// 因此它们与普通元素、游离文本一样被丢弃。
			name: "body content is dropped including head-style elements",
			raw: `<body><meta name="keywords" content="k"><link rel="stylesheet" href="/b.css">` +
				`<script src="/a.js"></script><style>a{}</style><title>t</title>` +
				`<template><meta name="x" content="y"></template><h1>hello</h1>text</body>`,
			want: "",
		},
		{
			// 实际存在的边界：head 合法元素写在 </head> 之后、body 内容之前时，
			// 解析器的 after-head 规则会把它归回 head，因此被保留。
			name: "head elements after the head end tag are reparsed into head",
			raw: "<head></head>" +
				`<meta name="keywords" content="k">` +
				`<link rel="preconnect" href="https://cdn.example.com">`,
			want: "<meta name=\"keywords\" content=\"k\"/>\n" +
				"<link rel=\"preconnect\" href=\"https://cdn.example.com\"/>",
		},
		{
			name: "loose text is dropped",
			raw:  "hello world",
			want: "",
		},
		{
			name: "empty input",
			raw:  "",
			want: "",
		},
		{
			name: "whitespace input",
			raw:  "  \n\t\n",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Sanitize(tc.raw)
			if got != tc.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tc.raw, got, tc.want)
			}
			if again := Sanitize(got); again != got {
				t.Errorf("Sanitize is not idempotent: Sanitize(%q) = %q, want %q", got, again, got)
			}
		})
	}
}
