package campusservice

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

const Issuer = "https://api.tongji.edu.cn/keycloak/realms/OpenPlatform"
const APIBase = "https://api.tongji.edu.cn"

var ErrDisabled = errors.New("campus_disabled")
var ErrAuthorization = errors.New("campus_authorization_required")
var ErrFlow = errors.New("campus_authorization_expired")
var ErrConflict = errors.New("campus_identity_unavailable")
var ErrUpstream = errors.New("campus_upstream_unavailable")
var ErrPermission = errors.New("campus_message_authorization_required")
var ErrMessageNotFound = errors.New("campus_message_unavailable")

type Config struct{ ClientID, ClientSecret, RedirectURI, EncryptionKey, IdentityKey string }

func configValue(name, env string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return preferences.GetString("campus." + name)
}
func Configured() (Config, error) {
	c := Config{configValue("clientId", "CAMPUS_CLIENT_ID"), configValue("clientSecret", "CAMPUS_CLIENT_SECRET"), configValue("redirectUri", "CAMPUS_REDIRECT_URI"), configValue("encryptionKey", "CAMPUS_ENCRYPTION_KEY"), configValue("identityKey", "CAMPUS_IDENTITY_KEY")}
	u, err := url.Parse(c.RedirectURI)
	if err != nil {
		return c, ErrDisabled
	}
	local := u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost")
	if u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/api/campus/tongji/callback" || (u.Scheme != "https" && !local) || c.ClientID == "" || len(c.EncryptionKey) < 32 || len(c.IdentityKey) < 32 {
		return c, ErrDisabled
	}

	return c, nil
}
func random() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func fingerprint(key, value string) string {
	h := hmac.New(sha256.New, []byte(key))
	_, _ = h.Write([]byte(Issuer + "\x00" + value))
	return hex.EncodeToString(h.Sum(nil))
}
func (c Config) cipher() (cipher.AEAD, error) {
	k := sha256.Sum256([]byte("yourtj-campus-v1\x00" + c.EncryptionKey))
	b, e := aes.NewCipher(k[:])
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(b)
}
func (c Config) seal(id uint64, v any) (string, error) {
	plain, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	a, e := c.cipher()
	if e != nil {
		return "", e
	}
	n := make([]byte, a.NonceSize())
	if _, e = rand.Read(n); e != nil {
		return "", e
	}
	return base64.RawStdEncoding.EncodeToString(a.Seal(n, n, plain, []byte(fmt.Sprint(id)))), nil
}
func (c Config) open(id uint64, v string, target any) error {
	a, e := c.cipher()
	if e != nil {
		return e
	}
	b, e := base64.RawStdEncoding.DecodeString(v)
	if e != nil || len(b) < a.NonceSize() {
		return ErrAuthorization
	}
	plain, e := a.Open(nil, b[:a.NonceSize()], b[a.NonceSize():], []byte(fmt.Sprint(id)))
	if e != nil {
		return ErrAuthorization
	}
	return json.Unmarshal(plain, target)
}

var serviceOnce sync.Once
var service atomic.Pointer[Service]

func Default() (*Service, error) {
	c, e := Configured()
	if e != nil {
		return nil, e
	}
	serviceOnce.Do(func() {
		instance := New(c, campus.Store{DB: dbconnect.Connect()}, NewProvider(c))
		instance.allowed = func(id uint64) bool {
			u, e := users.Get(id)
			return e == nil && u.Id != 0 && !u.IsBot() && !users.IsAccountClosed(id)
		}
		service.Store(instance)
	})
	return service.Load(), nil
}

// Account closure clears credentials even when integration configuration is disabled.
func CloseForUser(id uint64, closeAccount func() error) error {
	lock := &lifecycleLocks[id%64]
	lock.Lock()
	defer lock.Unlock()
	if instance := service.Load(); instance != nil {
		instance.dropPending(id)
	}
	if err := (campus.Store{DB: dbconnect.Connect()}).DeleteForUser(id); err != nil {
		return err
	}
	return closeAccount()
}
func Mask(value string) string {
	r := []rune(strings.TrimSpace(value))
	if len(r) < 5 {
		return "••••"
	}
	return string(r[:2]) + "••••" + string(r[len(r)-2:])
}
