package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestManagedBotAvatarUploadTargetsBotAndChargesHuman(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	if err := conn.AutoMigrate(&agents.Entity{}, &fileUsage.Entity{}); err != nil {
		t.Fatal(err)
	}
	owner := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, contractTestID())
	created, err := agentservice.Create(agentservice.CreateParams{Username: "avatar-managed-bot", CreatedBy: owner.Id})
	if err != nil {
		t.Fatal(err)
	}
	upload := func(actor *users.EntityComplete) *httptest.ResponseRecorder {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		if err := writer.WriteField("agentId", strconv.FormatUint(created.User.Id, 10)); err != nil {
			t.Fatal(err)
		}
		part, err := writer.CreateFormFile("avatar", "avatar.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(contractTinyPNG); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/upload-avatar", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+contractSessionToken(t, actor))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	denied := decodeContractEnvelope(t, upload(stranger))
	if denied.MessageCode != "permission.denied" {
		t.Fatalf("stranger allowed: %#v", denied)
	}
	rec := upload(owner)
	response := decodeContractEnvelope(t, rec)
	var result struct {
		AvatarUrl string `json:"avatarUrl"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("upload %s: %v", rec.Body.String(), err)
	}
	if !strings.HasPrefix(result.AvatarUrl, "/file/img/") {
		t.Fatalf("upload: %s", rec.Body.String())
	}
	name := strings.TrimPrefix(result.AvatarUrl, "/file/img/")
	file := filedata.GetByName(name)
	if file.UserId != owner.Id {
		t.Fatalf("file uploader = %d, want human %d", file.UserId, owner.Id)
	}
	otherBot, err := agentservice.Create(agentservice.CreateParams{Username: "avatar-other-bot", CreatedBy: stranger.Id})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := agentservice.Update(otherBot.User.Id, agentservice.UpdateParams{ActorID: stranger.Id, AvatarUrl: &result.AvatarUrl}); !errors.Is(err, agentservice.ErrAgentNicknameInvalid) {
		t.Fatalf("foreign avatar accepted: %v", err)
	}
	if _, err := agentservice.Update(created.User.Id, agentservice.UpdateParams{ActorID: owner.Id, AvatarUrl: &result.AvatarUrl}); err != nil {
		t.Fatalf("own served avatar rejected: %v", err)
	}

	admin := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(&admin).Update("role_id", 1).Error; err != nil {
		t.Fatal(err)
	}
	bio := "Updated by another manager"
	if _, err := agentservice.Update(created.User.Id, agentservice.UpdateParams{ActorID: admin.Id, AvatarUrl: &result.AvatarUrl, Bio: &bio}); err != nil {
		t.Fatalf("co-manager cannot retain the bot's current avatar: %v", err)
	}
	bot, err := users.Get(created.User.Id)
	if err != nil {
		t.Fatal(err)
	}
	human, err := users.Get(owner.Id)
	if err != nil {
		t.Fatal(err)
	}
	if bot.AvatarUrl != name || human.AvatarUrl == name {
		t.Fatal("avatar assigned to wrong identity")
	}
	var usage fileUsage.Entity
	if err := conn.Where("target_type = ? AND target_id = ? AND file_name = ? AND usage_type = ?", fileUsage.TargetUser, bot.Id, name, fileUsage.UsageAvatar).First(&usage).Error; err != nil {
		t.Fatal(err)
	}
}

func TestManagedBotProfileRejectsUnsafeAndSensitiveFields(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &moderationLog.Entity{}); err != nil {
		t.Fatal(err)
	}
	cfg := emptySecurityConfig()
	cfg.SensitiveWords = []string{"blocked-profile-term"}
	cfg.ReservedUsernames = []string{"administrator"}
	persistSecurityPolicyConfig(t, conn, cfg)
	created, err := agentservice.Create(agentservice.CreateParams{Username: "profile-policy-bot", Nickname: "Original", CreatedBy: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ field, value, code string }{
		{"bio", "blocked-profile-term", string(component.MessageContentSensitiveBlocked)},
		{"signature", "blocked-profile-term", string(component.MessageContentSensitiveBlocked)},
		{"nickname", "administrator", "auth.nickname.reserved"},
		{"profileCoverUrl", "javascript:alert(1)", string(component.MessageRequestInvalidParams)},
		{"avatarUrl", "https://example.com/arbitrary.png", string(component.MessageRequestInvalidParams)},
	} {
		body, err := json.Marshal(map[string]any{"agentId": created.User.Id, tc.field: tc.value})
		if err != nil {
			t.Fatal(err)
		}
		_, response := postAgent(t, router, "/api/admin/agent-update", string(body))
		if response["messageCode"] != tc.code {
			t.Fatalf("%s: %v", tc.field, response)
		}
	}
	view, err := agentservice.Get(created.User.Id)
	if err != nil {
		t.Fatal(err)
	}
	if view.User.Nickname != "Original" || view.User.Bio != "" || view.User.Signature != "" {
		t.Fatal("rejected profile mutated")
	}
}
