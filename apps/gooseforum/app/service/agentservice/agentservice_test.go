package agentservice

import (
	"errors"
	"strings"
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
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

func TestCreateAgentInvalidWebhookRejected(t *testing.T) {
	setupAgentTestDB(t)
	for _, raw := range []string{
		"ftp://example.com",
		"not-a-url",
		"https://",
		"http://localhost:8080/hook",
		"http://127.0.0.1/hook",
		"http://10.0.0.1/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/hook",
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
	// Re-enable restores access.
	if _, err := Update(result.Agent.UserId, UpdateParams{Enabled: ptr(int8(agents.StatusEnabled))}); err != nil {
		t.Fatalf("Update(enabled) error = %v", err)
	}
	if _, _, err := ResolveByToken(result.Token); err != nil {
		t.Fatalf("resolve after re-enable error = %v", err)
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
