// Package agentservice implements the admin-managed Agent lifecycle and the
// unique bearer token used to authenticate an Agent (bot persona).
//
// Tokens are only stored hashed; the plaintext is returned exclusively on
// create and rotate. The database keeps a short, non-secret prefix so a token
// can resolve its Agent row efficiently before the constant-time hash check.
package agentservice

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/userservice"
	"gorm.io/gorm"
)

const (
	// TokenMark is the recognizable prefix of every agent token.
	TokenMark = "agt_"
	// tokenRandomBytes is the entropy of the random part of a token (24 bytes
	// base64url-encoded = 32 chars).
	tokenRandomBytes = 24
	// tokenPrefixChars is how many random characters the non-secret prefix keeps.
	tokenPrefixChars = 8
	// maxNicknameRunes matches users.nickname varchar(64) across every DB.
	maxNicknameRunes = 64
	// maxWebhookLength bounds the stored webhook endpoint.
	maxWebhookLength = 512
	// lastUsedTouchInterval avoids a write on every authenticated request.
	lastUsedTouchInterval = time.Minute
)

var (
	ErrAgentNotFound        = errors.New("agentservice: agent not found")
	ErrAgentDisabled        = errors.New("agentservice: agent disabled")
	ErrAgentTokenInvalid    = errors.New("agentservice: invalid agent token")
	ErrAgentUsernameExists  = errors.New("agentservice: username already exists")
	ErrAgentNicknameInvalid = errors.New("agentservice: nickname invalid")
	ErrAgentWebhookInvalid  = errors.New("agentservice: webhook endpoint invalid")
	ErrAgentEnabledInvalid  = errors.New("agentservice: enabled status invalid")
)

// TokenPair carries the plaintext token plus the derived stored fields.
type TokenPair struct {
	Token  string
	Prefix string
	Hash   string
}

// GenerateToken creates a cryptographically random agent token.
func GenerateToken() (TokenPair, error) {
	randomBytes, err := algorithm.GenerateRandomBytes(tokenRandomBytes)
	if err != nil {
		return TokenPair{}, err
	}
	token := TokenMark + base64.RawURLEncoding.EncodeToString(randomBytes)
	prefix := token[:len(TokenMark)+tokenPrefixChars]
	return TokenPair{Token: token, Prefix: prefix, Hash: hashToken(token)}, nil
}

// hashToken returns the hex sha256 of a token. Tokens carry enough entropy
// (24 random bytes) that a fast hash is safe and deterministic.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// tokenPrefixOf extracts the indexed prefix from a raw token. Empty means the
// token does not have the expected shape.
func tokenPrefixOf(token string) string {
	if !strings.HasPrefix(token, TokenMark) || len(token) < len(TokenMark)+tokenPrefixChars {
		return ""
	}
	return token[:len(TokenMark)+tokenPrefixChars]
}

// ResolveByToken looks up the Agent owning token, verifies the hash in
// constant time, and returns the agent row plus its bot user. Disabled agents
// and non-bot users never resolve. On success the last-used timestamp is
// touched.
func ResolveByToken(token string) (*agents.Entity, *users.EntityComplete, error) {
	prefix := tokenPrefixOf(token)
	if prefix == "" {
		return nil, nil, ErrAgentTokenInvalid
	}
	agent := agents.GetByTokenPrefix(prefix)
	if agent == nil {
		return nil, nil, ErrAgentTokenInvalid
	}
	if agent.Enabled != agents.StatusEnabled {
		return nil, nil, ErrAgentDisabled
	}
	storedHash, err := hex.DecodeString(agent.TokenHash)
	if err != nil {
		return nil, nil, ErrAgentTokenInvalid
	}
	wantHash, err := hex.DecodeString(hashToken(token))
	if err != nil {
		return nil, nil, ErrAgentTokenInvalid
	}
	if subtle.ConstantTimeCompare(storedHash, wantHash) != 1 {
		return nil, nil, ErrAgentTokenInvalid
	}
	user, err := users.Get(agent.UserId)
	if err != nil || user.Id == 0 || !user.IsBot() || user.IsFrozen == users.StatusFrozen {
		return nil, nil, ErrAgentTokenInvalid
	}
	now := time.Now()
	if agent.LastUsedAt == nil || now.Sub(*agent.LastUsedAt) >= lastUsedTouchInterval {
		if err := agents.TouchLastUsedAt(agent.UserId, now); err != nil {
			slog.Warn("agent last-used update failed", "agentId", agent.UserId, "error", err)
		} else {
			agent.LastUsedAt = &now
		}
	}
	return agent, &user, nil
}

// CreateParams is the admin input for creating an Agent.
type CreateParams struct {
	Username        string
	Nickname        string
	WebhookEndpoint string
	CreatedBy       uint64
}

// CreateResult carries the persisted rows and the one-time plaintext token.
type CreateResult struct {
	Agent agents.Entity
	User  users.EntityComplete
	Token string
}

// Create creates the bot users row and the agents row atomically. The bot row
// has no email, no usable password, and no role; the token is shown exactly
// once in the returned result.
func Create(p CreateParams) (CreateResult, error) {
	username := strings.TrimSpace(p.Username)
	if username == "" {
		return CreateResult{}, errors.New("agentservice: username required")
	}
	if users.ExistUsername(username) {
		return CreateResult{}, ErrAgentUsernameExists
	}
	nickname, err := normalizeNickname(p.Nickname)
	if err != nil {
		return CreateResult{}, err
	}
	webhook, err := normalizeWebhookEndpoint(p.WebhookEndpoint)
	if err != nil {
		return CreateResult{}, err
	}
	tokenPair, err := GenerateToken()
	if err != nil {
		return CreateResult{}, err
	}

	now := time.Now()
	userEntity := users.EntityComplete{
		Username:    username,
		Nickname:    nickname,
		ActorType:   users.ActorTypeBot,
		IsActivated: users.ActivationSuccess,
		IsFrozen:    users.StatusNormal,
		ActivatedAt: &now,
		AvatarUrl:   users.RandAvatarUrl(),
	}
	agentEntity := agents.Entity{
		TokenPrefix:     tokenPair.Prefix,
		TokenHash:       tokenPair.Hash,
		WebhookEndpoint: webhook,
		Enabled:         agents.StatusEnabled,
		CreatedBy:       p.CreatedBy,
	}

	err = db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&userEntity).Error; err != nil {
			return err
		}
		agentEntity.UserId = userEntity.Id
		if err := tx.Create(&agentEntity).Error; err != nil {
			return err
		}
		stats := userStatistics.Entity{UserId: userEntity.Id}
		if err := tx.Create(&stats).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return CreateResult{}, ErrAgentUsernameExists
		}
		return CreateResult{}, err
	}
	return CreateResult{Agent: agentEntity, User: userEntity, Token: tokenPair.Token}, nil
}

// AgentView joins the agent row with its bot user for admin listing.
type AgentView struct {
	Agent agents.Entity
	User  users.EntityComplete
}

// List returns all agents with their bot users, newest first.
func List() []AgentView {
	agentEntities := agents.List()
	if len(agentEntities) == 0 {
		return nil
	}
	userIds := make([]uint64, 0, len(agentEntities))
	for _, entity := range agentEntities {
		userIds = append(userIds, entity.UserId)
	}
	userMap := users.GetMapByIds(userIds)
	views := make([]AgentView, 0, len(agentEntities))
	for _, entity := range agentEntities {
		if user, ok := userMap[entity.UserId]; ok {
			views = append(views, AgentView{Agent: *entity, User: *user})
		}
	}
	return views
}

// Get returns one agent with its bot user.
func Get(userID uint64) (*AgentView, error) {
	agent := agents.GetByUserID(userID)
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	user, err := users.Get(userID)
	if err != nil || user.Id == 0 || !user.IsBot() {
		return nil, ErrAgentNotFound
	}
	return &AgentView{Agent: *agent, User: user}, nil
}

// UpdateParams carries the mutable agent fields. Nil pointers leave a field
// unchanged.
type UpdateParams struct {
	Nickname        *string
	WebhookEndpoint *string
	Enabled         *int8
}

// Update changes the agent profile (nickname on the bot user), webhook
// endpoint, and/or enabled state atomically across both rows.
func Update(userID uint64, p UpdateParams) (*AgentView, error) {
	agentUpdates := make(map[string]any, 2)
	if p.WebhookEndpoint != nil {
		normalized, err := normalizeWebhookEndpoint(*p.WebhookEndpoint)
		if err != nil {
			return nil, err
		}
		agentUpdates["webhook_endpoint"] = normalized
	}
	if p.Enabled != nil {
		if *p.Enabled != agents.StatusDisabled && *p.Enabled != agents.StatusEnabled {
			return nil, ErrAgentEnabledInvalid
		}
		agentUpdates["enabled"] = *p.Enabled
	}

	var nickname *string
	if p.Nickname != nil {
		normalized, err := normalizeNickname(*p.Nickname)
		if err != nil {
			return nil, err
		}
		nickname = &normalized
	}

	if err := db.Connect().Transaction(func(tx *gorm.DB) error {
		if len(agentUpdates) > 0 {
			if err := agents.UpdateColumns(tx, userID, agentUpdates); err != nil {
				return err
			}
		} else if agents.GetByUserIDWithDB(tx, userID) == nil {
			return gorm.ErrRecordNotFound
		}
		if nickname != nil {
			result := tx.Model(&users.EntityComplete{}).
				Where("id = ? AND actor_type = ?", userID, users.ActorTypeBot).
				Update("nickname", *nickname)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				var count int64
				if err := tx.Model(&users.EntityComplete{}).
					Where("id = ? AND actor_type = ?", userID, users.ActorTypeBot).
					Count(&count).Error; err != nil {
					return err
				}
				if count == 0 {
					return gorm.ErrRecordNotFound
				}
			}
		}
		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	view, err := Get(userID)
	if err != nil {
		return nil, err
	}
	if nickname != nil {
		userservice.RefreshUserCaches(&view.User)
	}
	return view, nil
}

// RotateToken replaces the agent token. The old token stops resolving
// immediately because the stored prefix and hash are replaced atomically in
// one update. The new plaintext token is returned exactly once.
func RotateToken(userID uint64) (string, error) {
	tokenPair, err := GenerateToken()
	if err != nil {
		return "", err
	}
	if err := agents.UpdateToken(userID, tokenPair.Prefix, tokenPair.Hash); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrAgentNotFound
		}
		return "", err
	}
	return tokenPair.Token, nil
}

// Disable turns the agent off. Resolution is rejected while disabled.
func Disable(userID uint64) error {
	if err := agents.UpdateEnabled(userID, agents.StatusDisabled); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotFound
		}
		return err
	}
	return nil
}

func normalizeNickname(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if len([]rune(value)) > maxNicknameRunes {
		return "", ErrAgentNicknameInvalid
	}
	return value, nil
}

// normalizeWebhookEndpoint trims and validates the optional webhook endpoint.
// Empty is allowed. Stored endpoints must be HTTP(S), carry no credentials or
// fragment, and cannot target an obvious local/private literal. The webhook
// sender must repeat address validation after DNS resolution before dialing.
func normalizeWebhookEndpoint(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if len(value) > maxWebhookLength {
		return "", ErrAgentWebhookInvalid
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrAgentWebhookInvalid
	}
	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if hostname == "" || hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".internal") {
		return "", ErrAgentWebhookInvalid
	}
	if strings.Contains(hostname, "%") || looksLikeLegacyNumericIP(hostname) {
		return "", ErrAgentWebhookInvalid
	}
	if ip, err := netip.ParseAddr(hostname); err == nil && (!ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
		return "", ErrAgentWebhookInvalid
	}
	return value, nil
}

func looksLikeLegacyNumericIP(hostname string) bool {
	parts := strings.Split(hostname, ".")
	if len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		if !isLegacyNumericIPPart(part) {
			return false
		}
	}
	// Standard dotted-decimal addresses are handled by netip.ParseAddr. Any
	// other all-numeric form is a legacy inet_aton spelling and must not be
	// allowed to reach a future DNS/dial path.
	_, err := netip.ParseAddr(hostname)
	return err != nil
}

func isLegacyNumericIPPart(part string) bool {
	if part == "" {
		return false
	}
	if strings.HasPrefix(part, "0x") {
		if len(part) == 2 {
			return false
		}
		for _, r := range part[2:] {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return false
			}
		}
		return true
	}
	for _, r := range part {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
