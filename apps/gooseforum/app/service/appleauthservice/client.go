// Package appleauthservice implements native Sign in with Apple without trusting
// profile fields or identity assertions supplied by the mobile client.
package appleauthservice

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

const appleIssuer = "https://appleid.apple.com"

var ErrUnavailable = errors.New("Apple sign-in unavailable")
var ErrInvalidCredential = errors.New("invalid Apple credential")

// Config contains server-only Apple signing inputs. It must never be serialized.
type Config struct {
	ClientID, TeamID, KeyID string
	PrivateKey              *ecdsa.PrivateKey
}

type identity struct {
	Subject, RefreshToken string
}

type Client struct {
	config        Config
	http          *http.Client
	baseURL       string
	keysMu        sync.Mutex
	keys          map[string]*rsa.PublicKey
	keysUpdated   time.Time
	keysAttempted time.Time
}

func newClient(config Config, client *http.Client) *Client {
	return &Client{config: config, http: client, baseURL: appleIssuer}
}

// Authenticate checks the native token before consuming its one-use code, then
// verifies the server-to-server exchange token represents that same identity.
func (c *Client) Authenticate(ctx context.Context, code, token, nonce string) (identity, error) {
	if len(code) == 0 || len(code) > 4096 || len(token) == 0 || len(token) > 16384 || len(nonce) < 32 || len(nonce) > 128 {
		return identity{}, ErrInvalidCredential
	}
	expected := sha256.Sum256([]byte(nonce))
	subject, err := c.verify(ctx, token, hex.EncodeToString(expected[:]))
	if err != nil {
		return identity{}, err
	}
	secret, err := c.secret()
	if err != nil {
		return identity{}, ErrUnavailable
	}
	var response struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err = c.post(ctx, "/auth/token", url.Values{"client_id": {c.config.ClientID}, "client_secret": {secret}, "code": {code}, "grant_type": {"authorization_code"}}, &response); err != nil {
		return identity{}, err
	}
	exchanged, err := c.verify(ctx, response.IDToken, hex.EncodeToString(expected[:]))
	if err != nil || exchanged != subject || response.RefreshToken == "" || len(response.RefreshToken) > 16384 {
		return identity{}, ErrInvalidCredential
	}
	return identity{Subject: subject, RefreshToken: response.RefreshToken}, nil
}

func (c *Client) Revoke(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return ErrInvalidCredential
	}
	secret, err := c.secret()
	if err != nil {
		return ErrUnavailable
	}
	return c.post(ctx, "/auth/revoke", url.Values{"client_id": {c.config.ClientID}, "client_secret": {secret}, "token": {refreshToken}, "token_type_hint": {"refresh_token"}}, nil)
}

func (c *Client) secret() (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{Issuer: c.config.TeamID, Subject: c.config.ClientID, Audience: jwt.ClaimStrings{appleIssuer}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute))})
	token.Header["kid"] = c.config.KeyID
	return token.SignedString(c.config.PrivateKey)
}

func (c *Client) post(ctx context.Context, path string, values url.Values, result any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(values.Encode()))
	if err != nil {
		return ErrUnavailable
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.http.Do(request)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusUnauthorized {
			return ErrInvalidCredential
		}
		return ErrUnavailable
	}
	if result != nil {
		if err := json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(result); err != nil {
			return ErrUnavailable
		}
	}
	return nil
}

func (c *Client) verify(ctx context.Context, raw, nonce string) (string, error) {
	claims := struct {
		Nonce string `json:"nonce"`
		jwt.RegisteredClaims
	}{}
	parsed, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" || len(kid) > 128 {
			return nil, ErrInvalidCredential
		}
		return c.key(ctx, kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(appleIssuer), jwt.WithAudience(c.config.ClientID), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if errors.Is(err, ErrUnavailable) {
		return "", ErrUnavailable
	}
	if err != nil || !parsed.Valid || claims.IssuedAt == nil || claims.Subject == "" || len(claims.Subject) > 255 || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(nonce)) != 1 {
		return "", ErrInvalidCredential
	}
	return claims.Subject, nil
}

func (c *Client) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.keysMu.Lock()
	defer c.keysMu.Unlock()
	if key := c.keys[kid]; key != nil && time.Since(c.keysUpdated) < time.Hour {
		return key, nil
	}
	// Unknown kids cannot cause an unbounded stream of requests to Apple.
	if time.Since(c.keysAttempted) < time.Minute {
		return nil, ErrInvalidCredential
	}
	c.keysAttempted = time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/auth/keys", nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	response, err := c.http.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, ErrUnavailable
	}
	var set jose.JSONWebKeySet
	if err := json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(&set); err != nil {
		return nil, ErrUnavailable
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, entry := range set.Keys {
		key, ok := entry.Key.(*rsa.PublicKey)
		if ok && key.N.BitLen() >= 2048 && entry.KeyID != "" && entry.Algorithm == "RS256" && entry.Use == "sig" {
			keys[entry.KeyID] = key
		}
	}
	c.keys = keys
	c.keysUpdated = time.Now()
	if key := c.keys[kid]; key != nil {
		return key, nil
	}
	return nil, ErrInvalidCredential
}
