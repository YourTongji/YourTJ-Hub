package campusservice

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Only the scopes already used by OneTJ are requested. student_exams is not
// enabled for this client. Extra scopes must be verified before adding them.
const Scopes = "openid dc_user_student_info rt_onetongji_school_calendar_current_term_calendar rt_onetongji_student_timetable rt_onetongji_cet_score rt_onetongji_undergraduate_score rt_teaching_info_undergraduate_summarized_grades rt_teaching_info_sports_test_data rt_teaching_info_sports_test_health rt_onetongji_manual_arrange rt_onetongji_school_calendar_all_term_calendar rt_onetongji_msg_list rt_onetongji_msg_detail"

type Credentials struct {
	Access    string `json:"access_token"`
	Refresh   string `json:"refresh_token"`
	IDToken   string `json:"id_token,omitempty"`
	ExpiresIn int64  `json:"expires_in"`
	ExpiresAt int64  `json:"expiresAt"`
	Subject   string `json:"subject"`
	StudentID string `json:"studentId"`
}
type Provider interface {
	Authorize(state, nonce, verifier, mode string) string
	Exchange(context.Context, string, string) (Credentials, error)
	Identity(context.Context, Credentials, string) (Credentials, error)
	Refresh(context.Context, Credentials) (Credentials, error)
	Data(context.Context, string, string) (any, error)
	Message(context.Context, string, string) (any, error)
	Revoke(context.Context, Credentials)
}
type TongjiProvider struct {
	Config       Config
	Client       *http.Client
	Base, Issuer string
}

func NewProvider(c Config) *TongjiProvider {
	return &TongjiProvider{c, &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, APIBase, Issuer}
}
func (p *TongjiProvider) Authorize(state, nonce, verifier, mode string) string {
	h := sha256.Sum256([]byte(verifier))
	q := url.Values{"client_id": {p.Config.ClientID}, "redirect_uri": {p.Config.RedirectURI}, "response_type": {"code"}, "scope": {Scopes}, "state": {state}, "nonce": {nonce}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(h[:])}, "kc_idp_hint": {"tjiam"}}
	// A switch must actually offer a different identity instead of silently using SSO.
	if mode == "replace" || mode == "reauthorize" {
		q.Set("prompt", "login")
	}
	return p.Issuer + "/protocol/openid-connect/auth?" + q.Encode()
}
func (p *TongjiProvider) request(ctx context.Context, address, token string, form url.Values) (map[string]any, error) {
	method := http.MethodGet
	var body io.Reader
	if form != nil {
		method = http.MethodPost
		body = strings.NewReader(form.Encode())
	}
	req, e := http.NewRequestWithContext(ctx, method, address, body)
	if e != nil {
		return nil, ErrUpstream
	}
	req.Header.Set("Accept", "application/json")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, e := p.Client.Do(req)
	if e != nil {
		return nil, ErrUpstream
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode == http.StatusUnauthorized {
		return nil, ErrAuthorization
	}
	if res.StatusCode == http.StatusForbidden {
		return nil, ErrPermission
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, ErrUpstream
	}
	var result map[string]any
	decoder := json.NewDecoder(io.LimitReader(res.Body, 4<<20))
	decoder.UseNumber()
	if decoder.Decode(&result) != nil {
		return nil, ErrUpstream
	}
	// The gateway returns OAuth errors inside HTTP 200, including invalid_grant.
	if wrapped, ok := result["error_error"].(map[string]any); ok {
		var inner map[string]any
		_ = json.Unmarshal([]byte(str(wrapped["http_body"])), &inner)
		if inner["error"] == "invalid_grant" {
			return nil, ErrAuthorization
		}
		return nil, ErrUpstream
	}
	if result["error"] != nil {
		if result["error"] == "invalid_grant" {
			return nil, ErrAuthorization
		}
		return nil, ErrUpstream
	}
	return result, nil
}
func (p *TongjiProvider) token(ctx context.Context, form url.Values) (Credentials, error) {
	form.Set("client_id", p.Config.ClientID)
	if p.Config.ClientSecret != "" {
		form.Set("client_secret", p.Config.ClientSecret)
	}
	m, e := p.request(ctx, p.Base+"/v1/token", "", form)
	if e != nil {
		return Credentials{}, e
	}
	b, e := json.Marshal(m)
	if e != nil {
		return Credentials{}, ErrUpstream
	}
	var c Credentials
	if json.Unmarshal(b, &c) != nil || c.Access == "" || c.ExpiresIn <= 0 {
		return c, ErrUpstream
	}
	c.ExpiresAt = time.Now().Unix() + c.ExpiresIn
	return c, nil
}
func (p *TongjiProvider) Exchange(ctx context.Context, code, verifier string) (Credentials, error) {
	return p.token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {p.Config.RedirectURI}})
}

type identityClaims struct {
	jwt.RegisteredClaims
	Nonce string `json:"nonce"`
	AZP   string `json:"azp"`
}

func (p *TongjiProvider) Identity(ctx context.Context, c Credentials, nonce string) (Credentials, error) {
	keys, e := p.request(ctx, p.Issuer+"/protocol/openid-connect/certs", "", nil)
	if e != nil {
		return c, e
	}
	claims := &identityClaims{}
	t, e := jwt.ParseWithClaims(c.IDToken, claims, func(t *jwt.Token) (any, error) {
		list, _ := keys["keys"].([]any)
		for _, item := range list {
			k, _ := item.(map[string]any)
			if k["kid"] != t.Header["kid"] || k["kty"] != "RSA" || k["use"] != "sig" {
				continue
			}
			n, ne := base64.RawURLEncoding.DecodeString(str(k["n"]))
			ex, ee := base64.RawURLEncoding.DecodeString(str(k["e"]))
			if ne != nil || ee != nil || len(ex) > 4 || len(ex) == 0 || len(n) < 256 {
				return nil, ErrAuthorization
			}
			ev := new(big.Int).SetBytes(ex).Int64()
			if ev < 3 {
				return nil, ErrAuthorization
			}
			return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(ev)}, nil
		}
		return nil, ErrAuthorization
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(p.Issuer), jwt.WithAudience(p.Config.ClientID), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second))
	if e != nil || !t.Valid || claims.Subject == "" || claims.IssuedAt == nil || claims.Nonce != nonce || (claims.AZP != "" && claims.AZP != p.Config.ClientID) {
		return c, ErrAuthorization
	}
	info, e := p.request(ctx, p.Issuer+"/protocol/openid-connect/userinfo", c.Access, nil)
	if e != nil || info["sub"] != claims.Subject {
		return c, ErrAuthorization
	}
	data, e := p.Data(ctx, "identity", c.Access)
	if e != nil {
		return c, e
	}
	rows := objects(data)
	if len(rows) != 1 {
		return c, ErrAuthorization
	}
	id := str(rows[0]["userId"])
	if id == "" || (str(info["userid"]) != "" && str(info["userid"]) != id) {
		return c, ErrAuthorization
	}
	c.Subject = claims.Subject
	c.StudentID = id
	c.IDToken = ""
	return c, nil
}
func (p *TongjiProvider) Refresh(ctx context.Context, old Credentials) (Credentials, error) {
	if old.Refresh == "" {
		return old, ErrAuthorization
	}
	c, e := p.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {old.Refresh}})
	if e != nil {
		return old, e
	}
	info, e := p.request(ctx, p.Issuer+"/protocol/openid-connect/userinfo", c.Access, nil)
	if e != nil {
		return old, e
	}
	if info["sub"] != old.Subject {
		return old, ErrAuthorization
	}
	c.Subject = old.Subject
	c.StudentID = old.StudentID
	c.IDToken = ""
	if c.Refresh == "" {
		c.Refresh = old.Refresh
	}
	return c, nil
}

var datasets = map[string]string{
	"identity":     "/v1/dc/user/student_info",
	"profile":      "/v1/dc/user/student_info",
	"calendar":     "/v1/rt/onetongji/school_calendar_current_term_calendar",
	"timetable":    "/v1/rt/onetongji/student_timetable",
	"grades":       "/v1/rt/onetongji/undergraduate_score?calendarId=-1",
	"cet":          "/v1/rt/onetongji/cet_score",
	"summary":      "/v1/rt/teaching_info/undergraduate_summarized_grades",
	"sports":       "/v1/rt/teaching_info/sports_test_data",
	"health":       "/v1/rt/teaching_info/sports_test_health",
	"arrangements": "/v1/rt/onetongji/manual_arrange",
	"terms":        "/v1/rt/onetongji/school_calendar_all_term_calendar",
	"messages":     "/v1/rt/onetongji/msg_list",
}

func (p *TongjiProvider) Data(ctx context.Context, key, token string) (any, error) {
	path, ok := datasets[key]
	if !ok {
		return nil, ErrUpstream
	}
	return p.readData(ctx, path, token)
}

func (p *TongjiProvider) Message(ctx context.Context, id, token string) (any, error) {
	if !validMessageID(id) {
		return nil, ErrMessageNotFound
	}
	return p.readData(ctx, "/v1/rt/onetongji/msg_detail?"+url.Values{"id": {id}}.Encode(), token)
}

func (p *TongjiProvider) readData(ctx context.Context, path, token string) (any, error) {
	m, e := p.request(ctx, p.Base+path, token, nil)
	if e != nil {
		return nil, e
	}
	if m["code"] != "A00000" {
		return nil, ErrUpstream
	}
	return m["data"], nil
}

// Local unlink is authoritative. Keycloak's per-token RFC7009 revocation is
// best effort and never invokes realm logout or revokes other OneTJ sessions.
func (p *TongjiProvider) Revoke(ctx context.Context, c Credentials) {
	if c.Refresh == "" {
		return
	}
	f := url.Values{"client_id": {p.Config.ClientID}, "token": {c.Refresh}, "token_type_hint": {"refresh_token"}}
	if p.Config.ClientSecret != "" {
		f.Set("client_secret", p.Config.ClientSecret)
	}
	_, _ = p.request(ctx, p.Issuer+"/protocol/openid-connect/revoke", "", f)
}
