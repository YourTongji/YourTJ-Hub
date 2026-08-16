package pkservice

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// 一系统账号密码登录（SSO）链端点（对齐 YourTJCourse-Serverless
// backend/scripts/pk_crawler/utils/loginout.py 的 Login 流程）。
const (
	onesystemLoginEntryURL   = "https://1.tongji.edu.cn/api/ssoservice/system/loginIn"
	onesystemIAMIDPBase      = "https://iam.tongji.edu.cn/idp/"
	onesystemSessionLoginURL = "https://1.tongji.edu.cn/api/sessionservice/session/login"
	onesystemLoginUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	onesystemLoginMaxJump    = 10 // 登录前页面重定向链上限（防死循环）
)

var (
	cryptJSSrcRe   = regexp.MustCompile(`src="([^"]*crypt\.js[^"]*)"`)
	spAuthChainRe  = regexp.MustCompile(`"#spAuthChainCode1"[^']*'([^']+)'`)
	setPublicKeyRe = regexp.MustCompile(`encrypt\.setPublicKey\(\s*'([^']+)'`)
	loginFailedRe  = regexp.MustCompile(`(?is)<[A-Za-z0-9_:.-]*loginFailed[^>]*>\s*([^<]*?)\s*<`)
)

// onesystemLoginEndpoints 一系统登录链端点（可注入；测试用本地 httptest 覆盖）。
type onesystemLoginEndpoints struct {
	entryURL     string // 登录前页面入口（302 到 IAM AuthnEngine）
	idpBase      string // IAM IDP 基础 URL（crypt.js / AuthnEngine）
	sessionLogin string // 1.tongji 会话登录端点
}

// defaultOnesystemLoginEndpoints 一系统登录链端点（var 便于测试注入本地 httptest）。
var defaultOnesystemLoginEndpoints = func() onesystemLoginEndpoints {
	return onesystemLoginEndpoints{
		entryURL:     onesystemLoginEntryURL,
		idpBase:      onesystemIAMIDPBase,
		sessionLogin: onesystemSessionLoginURL,
	}
}

// newLoginHTTPClient 构造登录用 HTTP 客户端：忽略环境代理（等价上游 trust_env=False）、
// 不自动跟随重定向（302 链逐跳手动跟进，对齐 allow_redirects=False）、共享 cookie jar。
func newLoginHTTPClient() (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

// LoginAndGetCookie 用一系统学号/密码完成 SSO 登录，返回会话 Cookie header
// （与手动抓取浏览器 Cookie 等价，可直接用于同步管线）。
//
// 流程对齐 YourTJCourse-Serverless 的 Login 实现：
// loginIn 入口 → 提取 authnLcKey / spAuthChainCode / crypt.js → RSA(PKCS1v15) 加密密码
// → AuthnEngine 表单认证 → SSO 302 链 → sessionservice/session/login 换会话 Cookie。
//
// 触发「加强认证」（loginFailed!=false）时返回明确错误：本实现不内置 IMAP 验证码
// 读取（避免引入 go-imap 依赖，决策点），此时请改用 Cookie 凭证。
func LoginAndGetCookie(sno, password string) (string, error) {
	if strings.TrimSpace(sno) == "" || strings.TrimSpace(password) == "" {
		return "", errors.New("缺少一系统学号/密码")
	}
	client, err := newLoginHTTPClient()
	if err != nil {
		return "", err
	}
	return loginOnesystem(defaultOnesystemLoginEndpoints(), client, sno, password)
}

func loginOnesystem(ep onesystemLoginEndpoints, client *http.Client, sno, password string) (string, error) {
	// 第一步：登录前页面（跟随重定向到 IAM），拿到 CHAIN_URL/authnLcKey/spAuthChainCode/crypt.js。
	chainURL, pageHTML, err := followLoginGET(client, ep.entryURL, onesystemLoginMaxJump)
	if err != nil {
		return "", fmt.Errorf("一系统登录：打开登录页失败：%w", err)
	}
	authnLcKey := loginAuthnLcKey(chainURL)
	spAuthChainCode := ""
	if m := spAuthChainRe.FindStringSubmatch(string(pageHTML)); m != nil {
		spAuthChainCode = m[1]
	}
	cryptSrc := ""
	if m := cryptJSSrcRe.FindStringSubmatch(string(pageHTML)); m != nil {
		cryptSrc = m[1]
	}
	if authnLcKey == "" || spAuthChainCode == "" || cryptSrc == "" {
		return "", errors.New("一系统登录：登录页缺少 authnLcKey/spAuthChainCode/crypt.js（页面结构变化？）")
	}

	// 第二步：取 RSA 公钥并加密密码。cryptSrc 可能是相对路径（idpBase 拼接）或
	// 绝对路径（/static/...）乃至完整 URL；统一解析后再取。
	pub, err := fetchOnesystemRSAPublicKey(client, resolveIDPResourceURL(ep.idpBase, cryptSrc))
	if err != nil {
		return "", fmt.Errorf("一系统登录：获取 RSA 公钥失败：%w", err)
	}
	encPass, err := encryptOnesystemPassword(pub, password)
	if err != nil {
		return "", fmt.Errorf("一系统登录：密码加密失败：%w", err)
	}

	loginForm := url.Values{
		"j_username":      {sno},
		"j_password":      {encPass},
		"j_checkcode":     {"请输入验证码"},
		"op":              {"login"},
		"spAuthChainCode": {spAuthChainCode},
		"authnLcKey":      {authnLcKey},
	}

	// 第三步：预登录到 CHAIN_URL，检测「加强认证」。
	preResp, err := postLoginForm(client, chainURL, loginForm)
	if err != nil {
		return "", fmt.Errorf("一系统登录：预登录失败：%w", err)
	}
	preBody, _ := io.ReadAll(io.LimitReader(preResp.Body, 1<<20))
	preResp.Body.Close()
	if m := loginFailedRe.FindSubmatch(preBody); m != nil && strings.TrimSpace(string(m[1])) != "false" {
		return "", errors.New("一系统触发加强认证（需邮箱验证码）：当前版本未内置 IMAP 自动读码（决策点），请改用 Cookie 凭证（--onesystem-cookie / ONESYSTEM_COOKIE / 管理端设置）")
	}

	// 第四步：AuthnEngine 表单认证（非加强：BAMUsernamePassword）。
	authURL := ep.idpBase + "AuthnEngine?currentAuth=urn_oasis_names_tc_SAML_2.0_ac_classes_BAMUsernamePassword&authnLcKey=" +
		url.QueryEscape(authnLcKey) + "&entityId=SYS20230001"
	authResp, err := postLoginForm(client, authURL, loginForm)
	if err != nil {
		return "", fmt.Errorf("一系统登录：认证失败：%w", err)
	}
	authBody, _ := io.ReadAll(io.LimitReader(authResp.Body, 1<<20))
	authResp.Body.Close()
	loc := authResp.Header.Get("Location")
	if loc == "" {
		hint := strings.TrimSpace(string(authBody))
		if len(hint) > 200 {
			hint = hint[:200]
		}
		return "", fmt.Errorf("一系统登录：AuthnEngine 未返回跳转（账号/密码错误或已触发加强认证）：%s", redactCredentials(hint))
	}
	// AuthnEngine 的 Location 可能是相对路径（如 /sso?...），基于认证 URL 解析为绝对 URL。
	ssoURL := resolveLoginLocation(authURL, loc)

	// 第五步：SSO 302 链：sso → loginIn（code/state）→ ssologin（token/uid/ts）→ https 落地页。
	loginInURL, err := followLoginLocation(client, ssoURL)
	if err != nil {
		return "", fmt.Errorf("一系统登录：SSO 跳转失败：%w", err)
	}
	ssologinURL, err := followLoginLocation(client, loginInURL)
	if err != nil {
		return "", fmt.Errorf("一系统登录：取 ssologin 失败：%w", err)
	}
	httpsURL, err := followLoginLocation(client, ssologinURL)
	if err != nil {
		return "", fmt.Errorf("一系统登录：取落地页失败：%w", err)
	}
	if err := fetchLoginBodyNoCheck(client, httpsURL); err != nil {
		return "", fmt.Errorf("一系统登录：落地页失败：%w", err)
	}

	// 第六步：sessionservice/session/login 换会话 Cookie。
	token, uid, ts, err := loginSSOQueryParams(ssologinURL)
	if err != nil {
		return "", err
	}
	loginResp, err := postLoginForm(client, ep.sessionLogin, url.Values{"token": {token}, "ts": {ts}, "uid": {uid}})
	if err != nil {
		return "", fmt.Errorf("一系统登录：会话登录失败：%w", err)
	}
	loginBody, _ := io.ReadAll(io.LimitReader(loginResp.Body, 1<<20))
	loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("一系统登录：会话登录 HTTP %d：%s", loginResp.StatusCode, redactCredentials(strings.TrimSpace(string(loginBody))))
	}

	parsed, err := url.Parse(ep.sessionLogin)
	if err != nil {
		return "", err
	}
	cookies := client.Jar.Cookies(parsed)
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	if len(parts) == 0 {
		return "", errors.New("一系统登录：会话登录成功但未获得任何 Cookie")
	}
	return strings.Join(parts, "; "), nil
}

// loginAuthnLcKey 从最终登录页 URL 提取 authnLcKey；取不到时退化用最后一个 '=' 之后
// 的片段（对齐上游 response.url.split('=')[-1]）。
func loginAuthnLcKey(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil {
		if v := u.Query().Get("authnLcKey"); v != "" {
			return v
		}
	}
	if i := strings.LastIndex(rawURL, "="); i >= 0 && i+1 < len(rawURL) {
		return rawURL[i+1:]
	}
	return ""
}

// followLoginGET 手动跟随 GET 重定向链（对齐 allow_redirects=False 逐跳跟进），
// 返回最终 URL 与响应体；maxRedirects<=0 表示不限制（但每一步仍只取 Location）。
func followLoginGET(client *http.Client, rawURL string, maxRedirects int) (string, []byte, error) {
	cur := rawURL
	for i := 0; ; i++ {
		req, err := newLoginRequest(http.MethodGet, cur)
		if err != nil {
			return "", nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", nil, err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				return "", nil, fmt.Errorf("HTTP %d 无 Location", resp.StatusCode)
			}
			if maxRedirects > 0 && i >= maxRedirects {
				return "", nil, errors.New("重定向次数过多")
			}
			cur = resolveLoginLocation(cur, loc)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return "", nil, fmt.Errorf("HTTP %d：%s", resp.StatusCode, redactCredentials(strings.TrimSpace(string(body))))
		}
		return cur, body, nil
	}
}

// followLoginLocation 单跳 GET：返回响应的 Location（须为 3xx），不读 body。
func followLoginLocation(client *http.Client, rawURL string) (string, error) {
	req, err := newLoginRequest(http.MethodGet, rawURL)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d，期望 3xx 跳转", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", errors.New("3xx 响应无 Location")
	}
	return resolveLoginLocation(rawURL, loc), nil
}

// fetchLoginBodyNoCheck 单次 GET（不跟随重定向、不校验状态码），仅为推进 cookie jar
// 与完成落地（对齐上游 session.get(https_url, allow_redirects=False)）。
func fetchLoginBodyNoCheck(client *http.Client, rawURL string) error {
	req, err := newLoginRequest(http.MethodGet, rawURL)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	return nil
}

// postLoginForm 以表单编码 POST（不跟随重定向），带 UA/Origin/Content-Type 头。
func postLoginForm(client *http.Client, rawURL string, form url.Values) (*http.Response, error) {
	req, err := newLoginRequest(http.MethodPost, rawURL)
	if err != nil {
		return nil, err
	}
	encoded := form.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", originOf(rawURL))
	req.Body = io.NopCloser(strings.NewReader(encoded))
	req.ContentLength = int64(len(encoded))
	return client.Do(req)
}

func newLoginRequest(method, rawURL string) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", onesystemLoginUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	return req, nil
}

func resolveLoginLocation(base, loc string) string {
	u, err := url.Parse(loc)
	if err != nil || u.IsAbs() {
		return loc
	}
	b, err := url.Parse(base)
	if err != nil {
		return loc
	}
	return b.ResolveReference(u).String()
}

func originOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// resolveIDPResourceURL 解析 IDP 资源 URL：完整 URL 原样；绝对路径（/xxx）取 idpBase
// 的 scheme+host 拼接；相对路径拼到 idpBase 之后（对齐上游 idpBase + src 的假设）。
func resolveIDPResourceURL(idpBase, src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return idpBase
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src
	}
	if strings.HasPrefix(src, "/") {
		if u, err := url.Parse(idpBase); err == nil {
			u.Path = src
			u.RawQuery = ""
			return u.String()
		}
		return src
	}
	return idpBase + src
}

// fetchOnesystemRSAPublicKey 从 crypt.js 提取 RSA 公钥（encrypt.setPublicKey('...')）。
func fetchOnesystemRSAPublicKey(client *http.Client, jsURL string) (*rsa.PublicKey, error) {
	req, err := newLoginRequest(http.MethodGet, jsURL)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	m := setPublicKeyRe.FindSubmatch(body)
	if m == nil {
		return nil, errors.New("crypt.js 中未找到 encrypt.setPublicKey")
	}
	return parseOnesystemRSAPublicKey(string(m[1]))
}

// parseOnesystemRSAPublicKey 解析 RSA 公钥：接受 PEM、裸 base64 DER（PKIX 优先，退 PKCS1）。
func parseOnesystemRSAPublicKey(s string) (*rsa.PublicKey, error) {
	s = strings.TrimSpace(s)
	der := []byte(s)
	if strings.Contains(s, "BEGIN PUBLIC KEY") {
		block, _ := pem.Decode([]byte(s))
		if block == nil {
			return nil, errors.New("无法解析公钥 PEM")
		}
		der = block.Bytes
	} else {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("公钥 base64 解码失败：%w", err)
		}
		der = b
	}
	if pub, err := x509.ParsePKIXPublicKey(der); err == nil {
		if rsaPub, ok := pub.(*rsa.PublicKey); ok {
			return rsaPub, nil
		}
		return nil, errors.New("公钥不是 RSA 类型")
	}
	if pub, err := x509.ParsePKCS1PublicKey(der); err == nil {
		return pub, nil
	}
	return nil, errors.New("无法解析 RSA 公钥（PKIX/PKCS1 均失败）")
}

// encryptOnesystemPassword RSA(PKCS1v15) 加密密码后 base64（对齐上游 encryptPassword）。
func encryptOnesystemPassword(pub *rsa.PublicKey, password string) (string, error) {
	enc, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(enc), nil
}

// loginSSOQueryParams 从 ssologin URL 提取 token/uid/ts（query 优先，退化按位置：
// token&uid&ts，对齐上游 split('?')[1].split('&') 的假设）。
func loginSSOQueryParams(ssologinURL string) (token, uid, ts string, err error) {
	u, err := url.Parse(ssologinURL)
	if err != nil {
		return "", "", "", fmt.Errorf("解析 ssologin URL 失败：%w", err)
	}
	q := u.Query()
	token, uid, ts = q.Get("token"), q.Get("uid"), q.Get("ts")
	if token != "" && uid != "" && ts != "" {
		return token, uid, ts, nil
	}
	if raw := u.RawQuery; raw != "" {
		segs := strings.Split(raw, "&")
		if len(segs) >= 3 {
			pick := func(s string) string {
				_, v, _ := strings.Cut(s, "=")
				return v
			}
			if token == "" {
				token = pick(segs[0])
			}
			if uid == "" {
				uid = pick(segs[1])
			}
			if ts == "" {
				ts = pick(segs[2])
			}
		}
	}
	if token == "" || uid == "" || ts == "" {
		return "", "", "", errors.New("ssologin URL 缺少 token/uid/ts 参数")
	}
	return token, uid, ts, nil
}
