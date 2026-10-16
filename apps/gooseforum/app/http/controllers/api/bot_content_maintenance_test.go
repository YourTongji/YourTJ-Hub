package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/botcontentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"gorm.io/gorm"
)

func setupBotMaintenance(t *testing.T, id uint64) *gorm.DB {
	t.Helper()
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "")
	conn := setupTopicWriteTestDB(t)
	if err := conn.AutoMigrate(&agents.Entity{}, &optRecord.Entity{}, &taskQueue.Entity{}, &pageConfig.Entity{}, &moderationDecision.Entity{}, &eventNotification.Entity{}); err != nil {
		t.Fatal(err)
	}
	var previous pageConfig.Entity
	hadConfig := conn.Where("page_type = ?", pageConfig.SecuritySettings).First(&previous).Error == nil
	setSecurityConfigForCallbackTest(t, pageConfig.SecurityAndRegistration{})
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{})
		if hadConfig {
			conn.Create(&previous)
		}
		hotdataserve.ClearSecuritySettingsConfigCache()
	})
	for i := uint64(0); i < 4; i++ {
		createTopicWriteUser(t, conn, id+i, "maintenance")
	}
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", id+1).Updates(map[string]any{"role_id": 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", id+3).Updates(map[string]any{"actor_type": users.ActorTypeBot}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&agents.Entity{UserId: id + 3, CreatedBy: id, Enabled: agents.StatusEnabled, TokenPrefix: fmt.Sprintf("p%d", id), TokenHash: "unchanged", EventTypes: "[]"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&category.Entity{Id: id + 4, Name: "Maintenance", Slug: fmt.Sprintf("m%d", id)}).Error; err != nil {
		t.Fatal(err)
	}
	createRevisionTopic(t, conn, id+3, id+5, id+6, "original public bot content")
	if err := conn.Model(&posts.Entity{}).Where("id = ?", id+6).Update("content_type", posts.ContentTypeArticle).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", id+5).Update("category_id", fmt.Sprintf("[%d]", id+4)).Error; err != nil {
		t.Fatal(err)
	}
	return conn
}
func maintenanceVersion(v uint64) *uint64 { return &v }
func editBotTopic(actor, id uint64, version uint64) component.Response {
	return AgentEditTopic(component.BetterRequest[BotEditTopicReq]{UserId: actor, Params: BotEditTopicReq{TopicId: id + 5, Title: "Corrected bot title", Content: "corrected bot content with enough words", CategoryId: []uint64{id + 4}, ExpectedRevisionNo: maintenanceVersion(version)}})
}

func TestBotMaintenanceOwnershipAttributionAndConflict(t *testing.T) {
	const id = uint64(1173000)
	conn := setupBotMaintenance(t, id)
	for _, actor := range []uint64{id, id + 1} {
		response := AgentContentSource(component.BetterRequest[BotContentSourceReq]{UserId: actor, Params: BotContentSourceReq{TopicId: id + 5}})
		if response.Data.Code != component.SUCCESS {
			t.Fatalf("source actor %d: %+v", actor, response)
		}
	}
	if response := editBotTopic(id+2, id, 1); response.Code != http.StatusForbidden {
		t.Fatalf("stranger accepted: %+v", response)
	}
	if response := editBotTopic(id, id, 1); response.Data.Code != component.SUCCESS {
		t.Fatalf("owner edit: %+v", response)
	}
	post := posts.Get(id + 6)
	if post.UserId != id+3 || post.LastEditorId != id || post.Content != "corrected bot content with enough words" {
		t.Fatalf("post attribution: %+v", post)
	}
	revisions := postRevisions.ListByPostId(id + 6)
	if len(revisions) != 2 || revisions[1].EditorId != id {
		t.Fatalf("revisions: %+v", revisions)
	}
	var audit optRecord.Entity
	if err := conn.Where("opt_user_id = ? AND target_id = ?", id, fmt.Sprint(id+5)).First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audit.OptInfo, "admin.opt.agent.contentEdited") || !strings.Contains(audit.OptInfo, fmt.Sprint(id+3)) {
		t.Fatalf("audit: %+v", audit)
	}
	if response := editBotTopic(id+1, id, 1); response.Code != http.StatusConflict {
		t.Fatalf("stale edit accepted: %+v", response)
	}
	if response := editBotTopic(id+1, id, 2); response.Data.Code != component.SUCCESS {
		t.Fatalf("admin edit: %+v", response)
	}
	if got := posts.Get(id + 6).LastEditorId; got != id+1 {
		t.Fatalf("editor=%d", got)
	}
	var bot agents.Entity
	if err := conn.First(&bot, "user_id = ?", id+3).Error; err != nil {
		t.Fatal(err)
	}
	if bot.TokenHash != "unchanged" || bot.EventTypes != "[]" || bot.ConfigVersion != 0 {
		t.Fatalf("agent config changed: %+v", bot)
	}
}

func TestBotMaintenanceForbiddenSourceMatrix(t *testing.T) {
	for i, tc := range []struct {
		name  string
		model any
		patch map[string]any
	}{
		{"human", &users.EntityComplete{}, map[string]any{"actor_type": users.ActorTypeHuman}},
		{"frozen", &users.EntityComplete{}, map[string]any{"is_frozen": users.StatusFrozen}},
		{"disabled", &agents.Entity{}, map[string]any{"enabled": agents.StatusDisabled}},
		{"wiki", &topics.Entity{}, map[string]any{"topic_type": topics.TopicTypeWiki}},
		{"draft", &topics.Entity{}, map[string]any{"status": 0}},
		{"hidden", &topics.Entity{}, map[string]any{"visibility_status": topics.VisibilityModeratorRemoved}},
		{"blocked", &topics.Entity{}, map[string]any{"process_status": topics.ProcessStatusBlocked}},
		{"anonymous", &posts.Entity{}, map[string]any{"is_anonymous": true}},
		{"deletedPost", &posts.Entity{}, map[string]any{"visibility_status": posts.VisibilityUserDeleted}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uint64(1173100 + i*10)
			conn := setupBotMaintenance(t, id)
			key := id + 3
			field := "id"
			switch tc.model.(type) {
			case *agents.Entity:
				field = "user_id"
			case *topics.Entity:
				key = id + 5
			case *posts.Entity:
				key = id + 6
			}
			if err := conn.Model(tc.model).Where(field+" = ?", key).Updates(tc.patch).Error; err != nil {
				t.Fatal(err)
			}
			if response := editBotTopic(id+1, id, 1); response.Code != http.StatusForbidden {
				t.Fatalf("forbidden edit: %+v", response)
			}
			if got := posts.Get(id + 6).Content; got != "original public bot content" {
				t.Fatalf("changed source: %s", got)
			}
		})
	}
}

func TestBotMaintenanceReplyAndAtomicAuditFailure(t *testing.T) {
	const id = uint64(1173300)
	conn := setupBotMaintenance(t, id)
	createTopicReply(t, conn, id+7, id+5, 2, id+3, "original reply content")
	if err := conn.Create(&postRevisions.Entity{PostId: id + 7, Version: 1, EditorId: id + 3, Content: "original reply content"}).Error; err != nil {
		t.Fatal(err)
	}
	response := AgentEditPost(component.BetterRequest[BotEditPostReq]{UserId: id, Params: BotEditPostReq{PostId: id + 7, Content: "corrected reply content with enough words", ExpectedRevisionNo: maintenanceVersion(1)}})
	if response.Data.Code != component.SUCCESS {
		t.Fatalf("reply edit: %+v", response)
	}
	if post := posts.Get(id + 7); post.UserId != id+3 || post.LastEditorId != id {
		t.Fatalf("reply attribution: %+v", post)
	}
	callback := "maintenance_audit_failure"
	if err := conn.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "opt_record" {
			_ = tx.AddError(errors.New("audit unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove(callback) })
	if response := editBotTopic(id, id, 1); response.Data.Code != component.FAIL {
		t.Fatalf("audit failure ignored: %+v", response)
	}
	if post := posts.Get(id + 6); post.Content != "original public bot content" || post.LastEditorId != 0 {
		t.Fatalf("partial save: %+v", post)
	}
	if revisions := postRevisions.ListByPostId(id + 6); len(revisions) != 1 {
		t.Fatalf("partial revision: %d", len(revisions))
	}
}

func TestBotMaintenancePendingReviewPreservesPublicProjection(t *testing.T) {
	const id = uint64(1173400)
	conn := setupBotMaintenance(t, id)
	setSecurityConfigForCallbackTest(t, pageConfig.SecurityAndRegistration{SensitiveWords: []string{"review-marker"}, SensitiveAction: "review"})
	response := AgentEditTopic(component.BetterRequest[BotEditTopicReq]{UserId: id, Params: BotEditTopicReq{TopicId: id + 5, Title: "Corrected bot title", Content: "review-marker requires a review before publishing", CategoryId: []uint64{id + 4}, ExpectedRevisionNo: maintenanceVersion(1)}})
	if response.Data.Code != component.SUCCESS || response.Data.MessageCode != component.MessageContentModerationChecking {
		t.Fatalf("pending edit: %+v", response)
	}
	post := posts.Get(id + 6)
	if post.Content != "original public bot content" || post.UserId != id+3 || post.LastEditorId != id || post.LatestRevisionId == 0 {
		t.Fatalf("public projection lost: %+v", post)
	}
	sourceResponse := AgentContentSource(component.BetterRequest[BotContentSourceReq]{UserId: id, Params: BotContentSourceReq{TopicId: id + 5}})
	source, ok := sourceResponse.Data.Result.(BotContentSource)
	if !ok || source.Content != "review-marker requires a review before publishing" || source.Title != "Corrected bot title" {
		t.Fatalf("pending source is not latest submission: %+v", sourceResponse)
	}
	latest, err := postRevisions.LatestTx(conn, post.Id)
	if err != nil || latest.EditorId != id || latest.ProcessStatus != posts.ProcessStatusPending {
		t.Fatalf("candidate: %+v err=%v", latest, err)
	}
	var auditCount int64
	if err := conn.Model(&optRecord.Entity{}).Where("opt_user_id = ?", id).Count(&auditCount).Error; err != nil || auditCount != 1 {
		t.Fatalf("audit=%d %v", auditCount, err)
	}
}

func TestBotMaintenanceCommitRechecksDisablementAndOwnership(t *testing.T) {
	const id = uint64(1173500)
	conn := setupBotMaintenance(t, id)
	m, err := botcontentservice.Load(id, botcontentservice.SourceRequest{PostId: id + 6})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&agents.Entity{}).Where("user_id = ?", id+3).Update("enabled", agents.StatusDisabled).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(m.CheckTx); !errors.Is(err, botcontentservice.ErrUnavailable) {
		t.Fatalf("disabled commit allowed: %v", err)
	}
	if err := conn.Model(&agents.Entity{}).Where("user_id = ?", id+3).Updates(map[string]any{"enabled": agents.StatusEnabled, "created_by": id + 2}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(m.CheckTx); !errors.Is(err, botcontentservice.ErrUnavailable) {
		t.Fatalf("lost ownership allowed: %v", err)
	}
}

func TestBotMaintenancePendingApprovalRechecksManagerAndAgent(t *testing.T) {
	for i, change := range []string{"active", "disabled", "ownerChanged"} {
		t.Run(change, func(t *testing.T) {
			id := uint64(1173700 + i*10)
			conn := setupBotMaintenance(t, id)
			setSecurityConfigForCallbackTest(t, pageConfig.SecurityAndRegistration{SensitiveWords: []string{"review-marker"}, SensitiveAction: "review"})
			response := AgentEditTopic(component.BetterRequest[BotEditTopicReq]{UserId: id, Params: BotEditTopicReq{TopicId: id + 5, Title: "Corrected bot title", Content: "review-marker queued maintenance content", CategoryId: []uint64{id + 4}, ExpectedRevisionNo: maintenanceVersion(1)}})
			if response.Data.Code != component.SUCCESS {
				t.Fatalf("submit: %+v", response)
			}
			post := posts.Get(id + 6)
			switch change {
			case "disabled":
				if err := conn.Model(&agents.Entity{}).Where("user_id = ?", id+3).Update("enabled", 0).Error; err != nil {
					t.Fatal(err)
				}
			case "ownerChanged":
				if err := conn.Model(&agents.Entity{}).Where("user_id = ?", id+3).Update("created_by", id+2).Error; err != nil {
					t.Fatal(err)
				}
			}
			err := publicationservice.Review(context.Background(), post.LatestRevisionId, moderationDecision.ActionAllow, "", id+1)
			if change != "active" {
				if err != nil {
					t.Fatalf("revoked pending edit did not terminate: %v", err)
				}
				latest := postRevisions.Get(post.LatestRevisionId)
				if latest.ProcessStatus != posts.ProcessStatusBlocked || latest.ReviewedAt == nil {
					t.Fatalf("revocation not recorded: %+v", latest)
				}
				if got := posts.Get(post.Id).Content; got != "original public bot content" {
					t.Fatalf("public content replaced: %s", got)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				got := posts.Get(post.Id)
				if got.Content != "review-marker queued maintenance content" || got.UserId != id+3 || got.LastEditorId != id {
					t.Fatalf("approved projection: %+v", got)
				}
			}
		})
	}
}
