package passkeyservice

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPasskeys"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/totpservice"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ceremonyTTL         = 5 * time.Minute
	reauthenticationTTL = 2 * time.Minute
	maxCredentials      = 10
	maxAttempts         = 5
	credentialPurpose   = securestore.PasskeyCredentialPurpose
)

var (
	ErrDisabled       = errors.New("passkeys are unavailable")
	ErrInvalidRequest = errors.New("invalid passkey request")
	ErrInvalidProof   = errors.New("recent strong authentication is required")
	ErrLimitReached   = errors.New("passkey credential limit reached")
	ErrNotFound       = errors.New("passkey credential not found")
	ErrCounterWarning = errors.New("authenticator counter did not advance")
)

type RegistrationBeginResult struct {
	CeremonyID string                                      `json:"ceremonyId"`
	PublicKey  protocol.PublicKeyCredentialCreationOptions `json:"publicKey"`
}

type AuthenticationBeginResult struct {
	CeremonyID string                                     `json:"ceremonyId"`
	PublicKey  protocol.PublicKeyCredentialRequestOptions `json:"publicKey"`
}

type CredentialSummary struct {
	ID             string                            `json:"id"`
	Name           string                            `json:"name"`
	CreatedAt      time.Time                         `json:"createdAt"`
	LastUsedAt     *time.Time                        `json:"lastUsedAt,omitempty"`
	BackupEligible bool                              `json:"backupEligible"`
	BackupState    bool                              `json:"backupState"`
	Transports     []protocol.AuthenticatorTransport `json:"transports"`
}

type AuthenticatedSession struct {
	UserID uint64
	Token  string
	Jti    string
}

type rpUser struct {
	id          uint64
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u rpUser) WebAuthnID() []byte                         { return userHandle(u.id) }
func (u rpUser) WebAuthnName() string                       { return u.name }
func (u rpUser) WebAuthnDisplayName() string                { return u.displayName }
func (u rpUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func IsEnabled() (bool, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return false, err
	}
	return cfg.Enabled, nil
}

func Reauthenticate(userID uint64, sessionJti, password, totpCode string) (string, error) {
	if userID == 0 || sessionJti == "" || strings.TrimSpace(password) == "" {
		return "", ErrInvalidProof
	}
	proof, err := randomToken()
	if err != nil {
		return "", err
	}
	account, err := users.Get(userID)
	if err != nil || account.IsBot() || account.IsFrozen == users.StatusFrozen {
		return "", ErrInvalidProof
	}
	if algorithm.VerifyEncryptPassword(account.Password, password) != nil {
		return "", ErrInvalidProof
	}
	enabled, err := totpservice.IsEnabled(userID)
	if err != nil {
		return "", err
	}
	if enabled {
		valid, err := totpservice.Verify(userID, totpCode)
		if err != nil || !valid {
			return "", ErrInvalidProof
		}
	} else if totpCode != "" {
		return "", ErrInvalidProof
	}
	now := time.Now()
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		return userPasskeys.SaveChallenge(tx, &userPasskeys.ChallengeEntity{
			ID:         tokenDigest(proof),
			Kind:       "reauth",
			UserID:     userID,
			SessionJti: sessionJti,
			ExpiresAt:  now.Add(reauthenticationTTL),
		})
	})
	if err != nil {
		return "", err
	}
	return proof, nil
}

func BeginRegistration(userID uint64, sessionJti, proof string) (*RegistrationBeginResult, error) {
	cfg, err := LoadConfig()
	if err != nil || !cfg.Enabled {
		return nil, ErrDisabled
	}
	wa, err := cfg.NewWebAuthn()
	if err != nil {
		return nil, err
	}
	ceremonyID, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var result *RegistrationBeginResult
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		consumed, err := userPasskeys.ConsumeReauthenticationProof(tx, tokenDigest(proof), userID, sessionJti, now)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInvalidProof
		}
		account, err := loadWritableUser(tx, userID)
		if err != nil {
			return err
		}
		count, err := userPasskeys.CountActiveCredentials(tx, userID)
		if err != nil {
			return err
		}
		if count >= maxCredentials {
			return ErrLimitReached
		}
		credentials, err := loadCredentials(tx, userID)
		if err != nil {
			return err
		}
		user := makeRPUser(account, credentials)
		creation, session, err := wa.BeginRegistration(user, webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		}))
		if err != nil {
			return err
		}
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return err
		}
		if err := userPasskeys.CleanupExpired(tx, now); err != nil {
			return err
		}
		if err := userPasskeys.SaveChallenge(tx, &userPasskeys.ChallengeEntity{
			ID:          tokenDigest(ceremonyID),
			Kind:        "registration",
			UserID:      userID,
			SessionJti:  sessionJti,
			SessionData: string(sessionJSON),
			ExpiresAt:   now.Add(ceremonyTTL),
		}); err != nil {
			return err
		}
		result = &RegistrationBeginResult{CeremonyID: ceremonyID, PublicKey: creation.Response}
		return nil
	})
	return result, err
}

func FinishRegistration(userID uint64, sessionJti, ceremonyID, name string, request *http.Request) (*CredentialSummary, error) {
	cfg, err := LoadConfig()
	if err != nil || !cfg.Enabled {
		return nil, ErrDisabled
	}
	wa, err := cfg.NewWebAuthn()
	if err != nil {
		return nil, err
	}
	name, err = normalizeName(name)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var summary *CredentialSummary
	var ceremonyErr error
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		challenge, err := userPasskeys.FindChallenge(tx, tokenDigest(ceremonyID), "registration", true)
		if err != nil || challenge.UserID != userID || challenge.SessionJti != sessionJti {
			return ErrInvalidRequest
		}
		if challenge.Attempts >= maxAttempts {
			return ErrInvalidRequest
		}
		account, err := loadWritableUser(tx, userID)
		if err != nil {
			return err
		}
		count, err := userPasskeys.CountActiveCredentials(tx, userID)
		if err != nil {
			return err
		}
		if count >= maxCredentials {
			return ErrLimitReached
		}
		credentials, err := loadCredentials(tx, userID)
		if err != nil {
			return err
		}
		session, err := decodeSession(challenge.SessionData)
		if err != nil {
			return err
		}
		credential, finishErr := wa.FinishRegistration(makeRPUser(account, credentials), session, request)
		if finishErr != nil {
			ceremonyErr = ErrInvalidRequest
			return recordFailedAttempt(tx, challenge, now)
		}
		if len(credential.ID) == 0 || len(credential.ID) > 1024 {
			ceremonyErr = ErrInvalidRequest
			return recordFailedAttempt(tx, challenge, now)
		}
		encoded, err := encodeCredential(credential)
		if err != nil {
			return err
		}
		row := &userPasskeys.CredentialEntity{CredentialID: credential.ID, UserID: userID, CredentialData: encoded, Name: name}
		if err := tx.SavePoint("passkey_credential_insert").Error; err != nil {
			return err
		}
		if err := userPasskeys.CreateCredential(tx, row); err != nil {
			if isUniqueConstraint(err) {
				if rollbackErr := tx.RollbackTo("passkey_credential_insert").Error; rollbackErr != nil {
					return rollbackErr
				}
				ceremonyErr = ErrInvalidRequest
				return recordFailedAttempt(tx, challenge, now)
			}
			return err
		}
		consumed, err := userPasskeys.ConsumeChallenge(tx, challenge.ID, "registration", now)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInvalidRequest
		}
		summary = summarizeCredential(row, *credential)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ceremonyErr != nil {
		return nil, ceremonyErr
	}
	return summary, nil
}

func BeginAuthentication() (*AuthenticationBeginResult, error) {
	cfg, err := LoadConfig()
	if err != nil || !cfg.Enabled {
		return nil, ErrDisabled
	}
	wa, err := cfg.NewWebAuthn()
	if err != nil {
		return nil, err
	}
	ceremonyID, err := randomToken()
	if err != nil {
		return nil, err
	}
	options, session, err := wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return nil, err
	}
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		if err := userPasskeys.CleanupExpired(tx, now); err != nil {
			return err
		}
		return userPasskeys.SaveChallenge(tx, &userPasskeys.ChallengeEntity{
			ID:          tokenDigest(ceremonyID),
			Kind:        "authentication",
			SessionData: string(sessionJSON),
			ExpiresAt:   now.Add(ceremonyTTL),
		})
	})
	if err != nil {
		return nil, err
	}
	return &AuthenticationBeginResult{CeremonyID: ceremonyID, PublicKey: options.Response}, nil
}

func FinishAuthentication(ceremonyID string, request *http.Request, userAgent, ip string) (*AuthenticatedSession, error) {
	cfg, err := LoadConfig()
	if err != nil || !cfg.Enabled {
		return nil, ErrDisabled
	}
	wa, err := cfg.NewWebAuthn()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var result *AuthenticatedSession
	var ceremonyErr error
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		challenge, err := userPasskeys.FindChallenge(tx, tokenDigest(ceremonyID), "authentication", true)
		if err != nil {
			return ErrInvalidRequest
		}
		if challenge.Attempts >= maxAttempts {
			return ErrInvalidRequest
		}
		session, err := decodeSession(challenge.SessionData)
		if err != nil {
			return err
		}
		var account *users.EntityComplete
		var stored *userPasskeys.CredentialEntity
		var rpCredential webauthn.Credential
		handler := func(rawID, handle []byte) (webauthn.User, error) {
			stored, err = userPasskeys.FindActiveCredential(tx, rawID, true)
			if err != nil || stored.UserID == 0 {
				return nil, ErrInvalidRequest
			}
			if !hmac.Equal(userHandle(stored.UserID), handle) {
				return nil, ErrInvalidRequest
			}
			account, err = loadUser(tx, stored.UserID)
			if err != nil || account.IsBot() || account.IsFrozen == users.StatusFrozen {
				return nil, ErrInvalidRequest
			}
			rpCredential, err = decodeCredential(stored)
			if err != nil || !hmac.Equal(rpCredential.ID, rawID) {
				return nil, ErrInvalidRequest
			}
			return makeRPUser(account, []webauthn.Credential{rpCredential}), nil
		}
		authenticatedUser, updatedCredential, finishErr := wa.FinishPasskeyLogin(handler, session, request)
		if finishErr != nil || account == nil || stored == nil {
			ceremonyErr = ErrInvalidRequest
			return recordFailedAttempt(tx, challenge, now)
		}
		if updatedCredential.Authenticator.CloneWarning {
			ceremonyErr = ErrCounterWarning
			return recordFailedAttempt(tx, challenge, now)
		}
		resolved, ok := authenticatedUser.(rpUser)
		if !ok || resolved.id != account.Id || !hmac.Equal(resolved.WebAuthnID(), userHandle(account.Id)) {
			ceremonyErr = ErrInvalidRequest
			return recordFailedAttempt(tx, challenge, now)
		}
		updated, err := encodeCredential(updatedCredential)
		if err != nil {
			return err
		}
		if err := tx.Model(&userPasskeys.CredentialEntity{}).
			Where("credential_id = ? AND user_id = ? AND revoked_at IS NULL", updatedCredential.ID, account.Id).
			Updates(map[string]any{"credential_data": updated, "last_used_at": now}).Error; err != nil {
			return err
		}
		token, jti, err := jwtopt.CreateSessionToken(account.Id, account.TokenVersion)
		if err != nil {
			return err
		}
		if err := sessionservice.CreateWithDB(tx, account.Id, jti, userAgent, ip); err != nil {
			return err
		}
		consumed, err := userPasskeys.ConsumeChallenge(tx, challenge.ID, "authentication", now)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInvalidRequest
		}
		result = &AuthenticatedSession{UserID: account.Id, Token: token, Jti: jti}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ceremonyErr != nil {
		return nil, ceremonyErr
	}
	return result, nil
}

func ListCredentials(userID uint64) ([]CredentialSummary, error) {
	rows, err := userPasskeys.ListActiveCredentials(dbconnect.Connect(), userID)
	if err != nil {
		return nil, err
	}
	result := make([]CredentialSummary, 0, len(rows))
	for i := range rows {
		credential, err := decodeCredential(&rows[i])
		if err != nil {
			return nil, err
		}
		result = append(result, *summarizeCredential(&rows[i], credential))
	}
	return result, nil
}

func RenameCredential(userID uint64, sessionJti, proof, credentialID, name string) error {
	decodedID, err := DecodeCredentialID(credentialID)
	if err != nil {
		return ErrNotFound
	}
	name, err = normalizeName(name)
	if err != nil {
		return err
	}
	now := time.Now()
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		consumed, err := userPasskeys.ConsumeReauthenticationProof(tx, tokenDigest(proof), userID, sessionJti, now)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInvalidProof
		}
		changed, err := userPasskeys.RenameCredential(tx, userID, decodedID, name)
		if err != nil {
			return err
		}
		if !changed {
			return ErrNotFound
		}
		return nil
	})
}

func RevokeCredential(userID uint64, sessionJti, proof, credentialID string) error {
	decodedID, err := DecodeCredentialID(credentialID)
	if err != nil {
		return ErrNotFound
	}
	now := time.Now()
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		consumed, err := userPasskeys.ConsumeReauthenticationProof(tx, tokenDigest(proof), userID, sessionJti, now)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInvalidProof
		}
		changed, err := userPasskeys.RevokeCredential(tx, userID, decodedID, now)
		if err != nil {
			return err
		}
		if !changed {
			return ErrNotFound
		}
		return nil
	})
}

func DecodeCredentialID(value string) ([]byte, error) {
	if len(value) == 0 || len(value) > 1400 {
		return nil, ErrInvalidRequest
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) == 0 || len(decoded) > 1024 {
		return nil, ErrInvalidRequest
	}
	return decoded, nil
}

func encodeCredential(credential *webauthn.Credential) (string, error) {
	data, err := json.Marshal(credential)
	if err != nil {
		return "", err
	}
	return securestore.EncryptPurpose(string(data), credentialPurpose)
}

func decodeCredential(row *userPasskeys.CredentialEntity) (webauthn.Credential, error) {
	plain, err := securestore.DecryptPurpose(row.CredentialData, credentialPurpose)
	if err != nil {
		return webauthn.Credential{}, err
	}
	var credential webauthn.Credential
	if err := json.Unmarshal([]byte(plain), &credential); err != nil {
		return webauthn.Credential{}, err
	}
	if len(credential.ID) == 0 || !hmac.Equal(credential.ID, row.CredentialID) {
		return webauthn.Credential{}, errors.New("stored passkey credential ID mismatch")
	}
	return credential, nil
}

func loadCredentials(tx *gorm.DB, userID uint64) ([]webauthn.Credential, error) {
	rows, err := userPasskeys.ListActiveCredentials(tx, userID)
	if err != nil {
		return nil, err
	}
	credentials := make([]webauthn.Credential, 0, len(rows))
	for i := range rows {
		credential, err := decodeCredential(&rows[i])
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, credential)
	}
	return credentials, nil
}

func makeRPUser(account *users.EntityComplete, credentials []webauthn.Credential) rpUser {
	displayName := strings.TrimSpace(account.Nickname)
	if displayName == "" {
		displayName = account.Username
	}
	return rpUser{id: account.Id, name: account.Username, displayName: displayName, credentials: credentials}
}

func loadUser(tx *gorm.DB, userID uint64) (*users.EntityComplete, error) {
	var account users.EntityComplete
	if err := tx.Where("id = ?", userID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func loadWritableUser(tx *gorm.DB, userID uint64) (*users.EntityComplete, error) {
	var account users.EntityComplete
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&account).Error; err != nil {
		return nil, err
	}
	if account.IsBot() || account.IsFrozen == users.StatusFrozen || account.IsActivated != users.ActivationSuccess {
		return nil, ErrInvalidProof
	}
	return &account, nil
}

func decodeSession(raw string) (webauthn.SessionData, error) {
	var session webauthn.SessionData
	if raw == "" || json.Unmarshal([]byte(raw), &session) != nil || session.Challenge == "" || session.Expires.IsZero() {
		return webauthn.SessionData{}, ErrInvalidRequest
	}
	return session, nil
}

func summarizeCredential(row *userPasskeys.CredentialEntity, credential webauthn.Credential) *CredentialSummary {
	return &CredentialSummary{
		ID:             base64.RawURLEncoding.EncodeToString(row.CredentialID),
		Name:           row.Name,
		CreatedAt:      row.CreatedAt,
		LastUsedAt:     row.LastUsedAt,
		BackupEligible: credential.Flags.BackupEligible,
		BackupState:    credential.Flags.BackupState,
		Transports:     append([]protocol.AuthenticatorTransport{}, credential.Transport...),
	}
}

func recordFailedAttempt(tx *gorm.DB, challenge *userPasskeys.ChallengeEntity, now time.Time) error {
	if challenge.Attempts+1 >= maxAttempts {
		_, err := userPasskeys.ConsumeChallenge(tx, challenge.ID, challenge.Kind, now)
		return err
	}
	return userPasskeys.IncrementChallengeAttempt(tx, challenge.ID, challenge.Kind, maxAttempts)
}

func randomToken() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

func tokenDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", digest[:])
}

func userHandle(userID uint64) []byte {
	pepper, err := securestore.Pepper("yourtj-webauthn-user-handle-v1")
	if err != nil {
		return nil
	}
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], userID)
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write(id[:])
	return mac.Sum(nil)
}

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "Passkey"
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 80 {
		return "", ErrInvalidRequest
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", ErrInvalidRequest
		}
	}
	return value, nil
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "unique constraint") || strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}
