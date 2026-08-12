package agentservice

import (
	"errors"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupAgentTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &agents.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatalf("migrate agent tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func TestGenerateTokenShape(t *testing.T) {
	pair, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if !strings.HasPrefix(pair.Token, TokenMark) {
		t.Fatalf("token = %q, want %q prefix", pair.Token, TokenMark)
	}
	if len(pair.Token) != len(TokenMark)+32 {
		t.Fatalf("token length = %d, want %d", len(pair.Token), len(TokenMark)+32)
	}
	if pair.Prefix != pair.Token[:len(TokenMark)+tokenPrefixChars] {
		t.Fatalf("prefix = %q, want token head", pair.Prefix)
	}
	if pair.Hash == "" || pair.Hash == pair.Token {
		t.Fatalf("hash must be a non-identity digest")
	}
	if !strings.HasPrefix(pair.Prefix, TokenMark) {
		t.Fatalf("prefix = %q, want %q prefix", pair.Prefix, TokenMark)
	}
}

func TestGenerateTokenUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 8 {
		pair, err := GenerateToken()
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}
		if seen[pair.Token] {
			t.Fatal("token collision")
		}
		seen[pair.Token] = true
	}
}

func TestCreateAgentPersistsBotPersonaAndHashedToken(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{
		Username:        "agent-one",
		Nickname:        "Agent One",
		WebhookEndpoint: "https://example.com/hook",
		CreatedBy:       42,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Agent.UserId == 0 || result.User.Id != result.Agent.UserId {
		t.Fatalf("agent user id mismatch: agent=%d user=%d", result.Agent.UserId, result.User.Id)
	}
	user, err := users.Get(result.User.Id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if user.Id == 0 {
		t.Fatal("bot user missing")
	}
	if !user.IsBot() {
		t.Fatalf("actor_type = %d, want bot", user.ActorType)
	}
	if user.Email != "" {
		t.Fatalf("bot email = %q, want empty", user.Email)
	}
	if user.Password != "" {
		t.Fatal("bot password must be unusable (empty)")
	}
	if user.RoleId != 0 {
		t.Fatalf("bot roleId = %d, want 0", user.RoleId)
	}
	if user.IsActivated != users.ActivationSuccess {
		t.Fatalf("bot activation = %d, want activated", user.IsActivated)
	}
	if result.Agent.TokenHash == "" || result.Agent.TokenHash == result.Token {
		t.Fatal("token hash must be stored, not plaintext")
	}
	if !strings.HasPrefix(result.Agent.TokenPrefix, TokenMark) {
		t.Fatalf("token prefix = %q", result.Agent.TokenPrefix)
	}
	if result.Agent.WebhookEndpoint != "https://example.com/hook" {
		t.Fatalf("webhook = %q", result.Agent.WebhookEndpoint)
	}
	if result.Agent.Enabled != agents.StatusEnabled {
		t.Fatalf("enabled = %d, want 1", result.Agent.Enabled)
	}
	if result.Agent.CreatedBy != 42 {
		t.Fatalf("created_by = %d, want 42", result.Agent.CreatedBy)
	}
}

func TestCreateAgentDuplicateUsernameRejected(t *testing.T) {
	setupAgentTestDB(t)
	if _, err := Create(CreateParams{Username: "dup-agent"}); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	_, err := Create(CreateParams{Username: "dup-agent"})
	if !errors.Is(err, ErrAgentUsernameExists) {
		t.Fatalf("second Create() error = %v, want ErrAgentUsernameExists", err)
	}
}

func TestUsersUsernameHasDatabaseUniqueConstraint(t *testing.T) {
	setupAgentTestDB(t)
	first := users.EntityComplete{Username: "unique-user", ActorType: users.ActorTypeHuman}
	if err := db.Connect().Create(&first).Error; err != nil {
		t.Fatalf("create first user: %v", err)
	}
	duplicate := users.EntityComplete{Username: first.Username, ActorType: users.ActorTypeBot}
	if err := db.Connect().Create(&duplicate).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate username error = %v, want gorm.ErrDuplicatedKey", err)
	}
}

func TestCreateAndUpdateRejectOverlongNickname(t *testing.T) {
	setupAgentTestDB(t)
	tooLong := strings.Repeat("鹅", maxNicknameRunes+1)
	if _, err := Create(CreateParams{Username: "long-nickname-create", Nickname: tooLong}); !errors.Is(err, ErrAgentNicknameInvalid) {
		t.Fatalf("Create(overlong nickname) error = %v, want ErrAgentNicknameInvalid", err)
	}
	result, err := Create(CreateParams{Username: "long-nickname-update"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Update(result.Agent.UserId, UpdateParams{Nickname: &tooLong}); !errors.Is(err, ErrAgentNicknameInvalid) {
		t.Fatalf("Update(overlong nickname) error = %v, want ErrAgentNicknameInvalid", err)
	}
}

func TestCreateAgentInvalidWebhookRejected(t *testing.T) {
	setupAgentTestDB(t)
	for _, raw := range []string{
		"ftp://example.com",
		"not-a-url",
		"https://",
		"http://localhost:8080/hook",
		"http://127.0.0.1/hook",
		"http://127.1/hook",
		"http://0x7f000001/hook",
		"http://0x7f.1/hook",
		"http://127.0x0.1/hook",
		"http://0x7f.0x0.0x1/hook",
		"http://2130706433/hook",
		"http://10.0.0.1/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/hook",
		"http://[fe80::1%25eth0]/hook",
		"https://user:pass@example.com/hook",
		"https://example.com/hook#fragment",
	} {
		_, err := Create(CreateParams{Username: "bad-hook-" + strings.ReplaceAll(raw, ":", ""), WebhookEndpoint: raw})
		if !errors.Is(err, ErrAgentWebhookInvalid) {
			t.Fatalf("webhook %q error = %v, want ErrAgentWebhookInvalid", raw, err)
		}
	}
}

func TestResolveByTokenHappyPath(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "resolve-agent"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	agent, user, err := ResolveByToken(result.Token)
	if err != nil {
		t.Fatalf("ResolveByToken() error = %v", err)
	}
	if agent.UserId != result.Agent.UserId || user.Id != result.Agent.UserId {
		t.Fatalf("resolved ids mismatch: agent=%d user=%d want=%d", agent.UserId, user.Id, result.Agent.UserId)
	}
	if agent.LastUsedAt == nil {
		t.Fatal("last_used_at should be touched on resolve")
	}
}

func TestResolveByTokenRejectsFrozenBot(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "frozen-agent"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := db.Connect().Model(&users.EntityComplete{}).
		Where("id = ?", result.User.Id).
		Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatalf("freeze bot: %v", err)
	}
	if _, _, err := ResolveByToken(result.Token); !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("ResolveByToken(frozen) error = %v, want ErrAgentTokenInvalid", err)
	}
}

func TestResolveByTokenRejectsWrongToken(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "wrong-token"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// Same prefix, different random tail: hash comparison must fail.
	other, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	other.Token = result.Agent.TokenPrefix + other.Token[len(other.Token)-(len(other.Token)-len(result.Agent.TokenPrefix)):]
	if _, _, err := ResolveByToken(other.Token); !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("ResolveByToken(wrong) error = %v, want ErrAgentTokenInvalid", err)
	}
}

func TestResolveByTokenRejectsGarbage(t *testing.T) {
	setupAgentTestDB(t)
	if _, _, err := ResolveByToken(""); !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("ResolveByToken(empty) error = %v", err)
	}
	if _, _, err := ResolveByToken("not-a-token"); !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("ResolveByToken(garbage) error = %v", err)
	}
}

func TestRotateTokenInvalidatesOldToken(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "rotate-agent"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, _, err := ResolveByToken(result.Token); err != nil {
		t.Fatalf("pre-rotate resolve error = %v", err)
	}
	newToken, err := RotateToken(result.Agent.UserId)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}
	if newToken == result.Token {
		t.Fatal("rotated token must differ")
	}
	if _, _, err := ResolveByToken(result.Token); !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("old token after rotate error = %v, want invalid", err)
	}
	agent, _, err := ResolveByToken(newToken)
	if err != nil {
		t.Fatalf("new token resolve error = %v", err)
	}
	if agent.UserId != result.Agent.UserId {
		t.Fatalf("rotated agent id = %d, want %d", agent.UserId, result.Agent.UserId)
	}
}

func TestRotateTokenCASConflict(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "rotate-cas-agent"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// A stale prefix must not overwrite a concurrent rotation.
	other, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	affected, err := agents.UpdateTokenCAS(result.Agent.UserId, "stale-prefix", other.Prefix, other.Hash)
	if err != nil {
		t.Fatalf("UpdateTokenCAS(stale) error = %v", err)
	}
	if affected != 0 {
		t.Fatalf("CAS with stale prefix affected = %d, want 0", affected)
	}
	// The real rotation still works against the current prefix.
	newToken, err := RotateToken(result.Agent.UserId)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}
	if _, _, err := ResolveByToken(newToken); err != nil {
		t.Fatalf("new token resolve error = %v", err)
	}
}

func TestDisableInvalidatesAccess(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "disable-agent"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := Disable(result.Agent.UserId); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if _, _, err := ResolveByToken(result.Token); !errors.Is(err, ErrAgentDisabled) {
		t.Fatalf("resolve after disable error = %v, want ErrAgentDisabled", err)
	}
	stored := agents.GetByUserID(result.Agent.UserId)
	if stored == nil || stored.TokenHash != "" {
		t.Fatal("disable must revoke the stored token hash")
	}
	// Re-enabling without an explicit rotation is rejected.
	if _, err := Update(result.Agent.UserId, UpdateParams{Enabled: ptr(int8(agents.StatusEnabled))}); !errors.Is(err, ErrAgentNeedsRotate) {
		t.Fatalf("re-enable after disable error = %v, want ErrAgentNeedsRotate", err)
	}
	// Rotation is the recovery path: the new token resolves after re-enable.
	newToken, err := RotateToken(result.Agent.UserId)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}
	if _, err := Update(result.Agent.UserId, UpdateParams{Enabled: ptr(int8(agents.StatusEnabled))}); err != nil {
		t.Fatalf("re-enable after rotation error = %v", err)
	}
	if _, _, err := ResolveByToken(newToken); err != nil {
		t.Fatalf("resolve new token after re-enable error = %v", err)
	}
	if _, _, err := ResolveByToken(result.Token); !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("old revoked token error = %v, want invalid", err)
	}
}

func TestAgentSecurityUpdatesPreserveUnownedColumns(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "security-columns", WebhookEndpoint: "https://before.example.com/hook"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	newToken, err := RotateToken(result.Agent.UserId)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}
	rotated := agents.GetByUserID(result.Agent.UserId)
	if rotated == nil {
		t.Fatal("rotated agent missing")
	}
	if err := Disable(result.Agent.UserId); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if _, err := Update(result.Agent.UserId, UpdateParams{WebhookEndpoint: ptr("https://after.example.com/hook")}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	stored := agents.GetByUserID(result.Agent.UserId)
	if stored == nil {
		t.Fatal("stored agent missing")
	}
	if stored.Enabled != agents.StatusDisabled {
		t.Fatalf("enabled = %d, want disabled", stored.Enabled)
	}
	if stored.TokenHash != "" {
		t.Fatal("disable must revoke the token hash")
	}
	if stored.TokenPrefix != rotated.TokenPrefix {
		t.Fatal("disable must retain the non-secret token prefix")
	}
	if stored.WebhookEndpoint != "https://after.example.com/hook" {
		t.Fatalf("webhook = %q", stored.WebhookEndpoint)
	}
	if _, _, err := ResolveByToken(result.Token); !errors.Is(err, ErrAgentDisabled) && !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("old token error = %v, want disabled or invalid", err)
	}
	// Re-enabling without rotation is rejected.
	if _, err := Update(result.Agent.UserId, UpdateParams{Enabled: ptr(int8(agents.StatusEnabled))}); !errors.Is(err, ErrAgentNeedsRotate) {
		t.Fatalf("re-enable after disable error = %v, want ErrAgentNeedsRotate", err)
	}
	// Rotate then re-enable restores access with the new token only.
	rotatedToken, err := RotateToken(result.Agent.UserId)
	if err != nil {
		t.Fatalf("RotateToken() after disable error = %v", err)
	}
	if _, err := Update(result.Agent.UserId, UpdateParams{Enabled: ptr(int8(agents.StatusEnabled))}); err != nil {
		t.Fatalf("re-enable after rotation error = %v", err)
	}
	if _, _, err := ResolveByToken(rotatedToken); err != nil {
		t.Fatalf("rotated token after re-enable error = %v", err)
	}
	if _, _, err := ResolveByToken(newToken); !errors.Is(err, ErrAgentTokenInvalid) {
		t.Fatalf("pre-disable token error = %v, want invalid", err)
	}
}

func TestUpdateProfileAndWebhook(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "update-agent", Nickname: "Before"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	view, err := Update(result.Agent.UserId, UpdateParams{
		Nickname:        ptr("After"),
		WebhookEndpoint: ptr("https://new.example.com/hook"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if view.User.Nickname != "After" {
		t.Fatalf("nickname = %q, want After", view.User.Nickname)
	}
	if view.Agent.WebhookEndpoint != "https://new.example.com/hook" {
		t.Fatalf("webhook = %q", view.Agent.WebhookEndpoint)
	}
	stored := agents.GetByUserID(result.Agent.UserId)
	if stored == nil || stored.WebhookEndpoint != "https://new.example.com/hook" {
		t.Fatal("webhook not persisted")
	}
	// Invalid webhook rejected without mutating.
	if _, err := Update(result.Agent.UserId, UpdateParams{WebhookEndpoint: ptr("ftp://x")}); !errors.Is(err, ErrAgentWebhookInvalid) {
		t.Fatalf("Update(bad webhook) error = %v, want ErrAgentWebhookInvalid", err)
	}

	if _, err := Update(result.Agent.UserId, UpdateParams{Enabled: ptr(int8(2))}); !errors.Is(err, ErrAgentEnabledInvalid) {
		t.Fatalf("Update(invalid enabled) error = %v, want ErrAgentEnabledInvalid", err)
	}
}

func TestAgentUpdatesAreIdempotent(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "idempotent-agent", Nickname: "Same", WebhookEndpoint: "https://same.example.com/hook"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Update(result.Agent.UserId, UpdateParams{
		Nickname:        ptr("Same"),
		WebhookEndpoint: ptr("https://same.example.com/hook"),
		Enabled:         ptr(int8(agents.StatusEnabled)),
	}); err != nil {
		t.Fatalf("Update(no-op) error = %v", err)
	}
	if err := Disable(result.Agent.UserId); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if err := Disable(result.Agent.UserId); err != nil {
		t.Fatalf("Disable(no-op) error = %v", err)
	}
}

func TestAgentUpdateRefreshesUpdatedAt(t *testing.T) {
	setupAgentTestDB(t)
	result, err := Create(CreateParams{Username: "updated-at-agent"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	baseline := time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC)
	if err := db.Connect().Model(&agents.Entity{}).
		Where("user_id = ?", result.Agent.UserId).
		UpdateColumn("updated_at", baseline).Error; err != nil {
		t.Fatalf("set baseline updated_at: %v", err)
	}
	if _, err := Update(result.Agent.UserId, UpdateParams{
		WebhookEndpoint: ptr("https://updated.example.com/hook"),
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	stored := agents.GetByUserID(result.Agent.UserId)
	if stored == nil {
		t.Fatal("stored agent missing")
	}
	if !stored.UpdatedAt.After(baseline) {
		t.Fatalf("updated_at = %v, want after %v", stored.UpdatedAt, baseline)
	}
}

func TestGetAndListAgents(t *testing.T) {
	setupAgentTestDB(t)
	first, err := Create(CreateParams{Username: "list-one"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	second, err := Create(CreateParams{Username: "list-two"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	views := List()
	if len(views) != 2 {
		t.Fatalf("List() len = %d, want 2", len(views))
	}
	// Newest first.
	if views[0].User.Username != "list-two" || views[1].User.Username != "list-one" {
		t.Fatalf("List() order = %q, %q", views[0].User.Username, views[1].User.Username)
	}
	view, err := Get(first.Agent.UserId)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if view.User.Username != "list-one" {
		t.Fatalf("Get() username = %q", view.User.Username)
	}
	if _, err := Get(second.Agent.UserId + 999); !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrAgentNotFound", err)
	}
}

func ptr[T any](v T) *T {
	return &v
}
