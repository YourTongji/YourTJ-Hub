package pkservice

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// loginPageHTML 模拟一系统登录前页面：内含 crypt.js 引用与 spAuthChainCode。
func loginPageHTML(cryptPath string) string {
	return `<html><head><script src="` + cryptPath + `"></script></head><body>
<script>$("#spAuthChainCode1").val('4c1eb8ec14fa4e8ba0f31188dbf88cdd');</script>
</body></html>`
}

// newLoginFixture 构造一条完整的一系统登录链 mock:
// entry(302→iam) → iam(登录页+crypt.js) → chain(POST 预登录) → AuthnEngine(POST→302 sso)
// → sso(302 loginIn) → loginIn(302 ssologin) → ssologin(302 https) → https(200)
// → session/login(POST 200, 种会话 cookie)。
// 返回 endpoints、模拟服务与断言函数（校验密码被 RSA 加密后能解密为明文）。
func newLoginFixture(t *testing.T, password string) (onesystemLoginEndpoints, func(t *testing.T)) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	pub := &priv.PublicKey
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal pubkey: %v", err)
	}
	cryptJS := "window.encrypt = {}; encrypt.setPublicKey('" + base64.StdEncoding.EncodeToString(der) + "');\n"

	var captured url.Values
	assertEncrypted := func(t *testing.T) {
		t.Helper()
		enc, err := base64.StdEncoding.DecodeString(captured.Get("j_password"))
		if err != nil {
			t.Fatalf("j_password is not base64: %v", err)
		}
		plain, err := rsa.DecryptPKCS1v15(rand.Reader, priv, enc)
		if err != nil {
			t.Fatalf("decrypt j_password: %v", err)
		}
		if string(plain) != password {
			t.Fatalf("j_password plaintext = %q, want %q", plain, password)
		}
		if captured.Get("j_username") == "" || captured.Get("spAuthChainCode") != "4c1eb8ec14fa4e8ba0f31188dbf88cdd" {
			t.Fatalf("login form missing fields: %v", captured)
		}
	}

	mux := http.NewServeMux()
	// 登录前入口：302 到 IAM 登录页（authnLcKey 带在 URL 上）。
	mux.HandleFunc("/entry", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/iam/login?authnLcKey=abc123", http.StatusFound)
	})
	// IAM 登录页：GET 返回登录页（含 crypt.js 与 spAuthChainCode）；
	// POST 为预登录（chain），返回加强认证判定（loginFailed=false）。
	mux.HandleFunc("/iam/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			captured = r.PostForm
			w.Header().Set("Content-Type", "text/xml")
			fmt.Fprint(w, `<SAML><loginFailed>false</loginFailed></SAML>`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, loginPageHTML("/iam/crypt.js"))
	})
	// crypt.js：RSA 公钥。
	mux.HandleFunc("/iam/crypt.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write([]byte(cryptJS))
	})
	// AuthnEngine POST：302 到 sso（带 token/uid/ts）。
	mux.HandleFunc("/iam/AuthnEngine", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("AuthnEngine method = %s, want POST", r.Method)
		}
		_ = r.ParseForm()
		if r.PostForm.Get("j_username") == "" || r.PostForm.Get("j_password") == "" {
			http.Error(w, "missing form", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/sso?token=tok&uid=u&ts=123", http.StatusFound)
	})
	// sso → loginIn → ssologin → https 的 302 链。
	mux.HandleFunc("/sso", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loginIn?code=c&state=s", http.StatusFound)
	})
	mux.HandleFunc("/loginIn", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ssologin?token=tok&uid=u&ts=123", http.StatusFound)
	})
	mux.HandleFunc("/ssologin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/https-landing", http.StatusFound)
	})
	mux.HandleFunc("/https-landing", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><script src="/static/js/app.abc.js"></script></html>`)
	})
	// session/login POST：种会话 cookie。
	mux.HandleFunc("/session/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("session/login method = %s, want POST", r.Method)
		}
		_ = r.ParseForm()
		if r.PostForm.Get("token") != "tok" || r.PostForm.Get("uid") != "u" || r.PostForm.Get("ts") != "123" {
			http.Error(w, "bad sso params", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "JWTUser", Value: "jwt"})
		http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "sess"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":0}`)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ep := onesystemLoginEndpoints{
		entryURL:     srv.URL + "/entry",
		idpBase:      srv.URL + "/iam/",
		sessionLogin: srv.URL + "/session/login",
	}
	return ep, assertEncrypted
}

func TestLoginAndGetCookieSuccess(t *testing.T) {
	ep, assertEncrypted := newLoginFixture(t, "s3cret-pass")
	client, err := newLoginHTTPClient()
	if err != nil {
		t.Fatalf("newLoginHTTPClient: %v", err)
	}
	cookie, err := loginOnesystem(ep, client, "1951234", "s3cret-pass")
	if err != nil {
		t.Fatalf("loginOnesystem: %v", err)
	}
	assertEncrypted(t)
	if !strings.Contains(cookie, "JWTUser=jwt") || !strings.Contains(cookie, "JSESSIONID=sess") {
		t.Fatalf("cookie = %q, want JWTUser and JSESSIONID", cookie)
	}
}

func TestLoginOnesystemEnhanceAuthFailsWithClearError(t *testing.T) {
	ep, _ := newLoginFixture(t, "pw")
	// 篡改 chain 响应为加强认证。
	ep2 := ep
	// 用全新 fixture 改不了 mux；单独起一个「加强认证」场景的 mock 更清晰。
	_ = ep2

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	pub := &priv.PublicKey
	der, _ := x509.MarshalPKIXPublicKey(pub)
	cryptJS := "encrypt.setPublicKey('" + base64.StdEncoding.EncodeToString(der) + "');\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/entry", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/iam/login?authnLcKey=abc", http.StatusFound)
	})
	mux.HandleFunc("/iam/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/xml")
			fmt.Fprint(w, `<SAML><loginFailed>true</loginFailed></SAML>`)
			return
		}
		fmt.Fprint(w, loginPageHTML("/iam/crypt.js"))
	})
	mux.HandleFunc("/iam/crypt.js", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(cryptJS))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client, err := newLoginHTTPClient()
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	epE := onesystemLoginEndpoints{entryURL: srv.URL + "/entry", idpBase: srv.URL + "/iam/", sessionLogin: srv.URL + "/session/login"}
	_, err = loginOnesystem(epE, client, "1951234", "pw")
	if err == nil {
		t.Fatal("want error for enhance auth, got nil")
	}
	if !strings.Contains(err.Error(), "加强认证") {
		t.Fatalf("error should mention enhance auth, got: %v", err)
	}
}

func TestLoginSSOQueryParams(t *testing.T) {
	token, uid, ts, err := loginSSOQueryParams("https://1.tongji.edu.cn/ssologin?token=tok&uid=u&ts=123")
	if err != nil || token != "tok" || uid != "u" || ts != "123" {
		t.Fatalf("query params = %q/%q/%q err=%v", token, uid, ts, err)
	}
	// 退化：query 键名变化时按位置取（对齐上游 split('&')）。
	token, uid, ts, err = loginSSOQueryParams("https://1.tongji.edu.cn/ssologin?tok=1&u=2&t=3")
	if err != nil || token != "1" || uid != "2" || ts != "3" {
		t.Fatalf("fallback params = %q/%q/%q err=%v", token, uid, ts, err)
	}
	if _, _, _, err := loginSSOQueryParams("https://1.tongji.edu.cn/x?a=1"); err == nil {
		t.Fatal("want error for missing params")
	}
}

func TestParseOnesystemRSAPublicKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// PEM 形式。
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if _, err := parseOnesystemRSAPublicKey(string(pemKey)); err != nil {
		t.Fatalf("parse PEM: %v", err)
	}
	// 裸 base64 DER。
	if _, err := parseOnesystemRSAPublicKey(base64.StdEncoding.EncodeToString(der)); err != nil {
		t.Fatalf("parse base64: %v", err)
	}
	// 非法输入。
	if _, err := parseOnesystemRSAPublicKey("not-a-key"); err == nil {
		t.Fatal("want error for invalid key")
	}
}

func TestLoginAuthnLcKey(t *testing.T) {
	if got := loginAuthnLcKey("https://iam.tongji.edu.cn/idp/AuthnEngine?x=1&authnLcKey=abc123"); got != "abc123" {
		t.Fatalf("authnLcKey = %q, want abc123", got)
	}
	if got := loginAuthnLcKey("https://1.tongji.edu.cn/x?k=val"); got != "val" {
		t.Fatalf("fallback authnLcKey = %q, want val", got)
	}
}
